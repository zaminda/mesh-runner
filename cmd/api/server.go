package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zaminda/mesh-runner/internal"
	"github.com/zaminda/mesh-runner/internal/jobstore"
)

type server struct {
	store    *jobstore.Store
	natsConn *nats.Conn
}

type createJobRequest struct {
	Command string `json:"command"`
}

type createJobResponse struct {
	ID        int       `json:"id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type jobSummary struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type listJobsResponse struct {
	Jobs []jobSummary `json:"jobs"`
}

func newServer() *server {
	natsURL := os.Getenv("NATS_URL")

	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	conn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	return &server{
		store:    jobstore.New(),
		natsConn: conn,
	}
}

func (s *server) listenAndServe() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:           addr,
		Handler:        s.routes(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
		close(done)
	}()

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	<-done
	log.Println("server stopped")
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.logRequest(s.handleHealth))
	mux.HandleFunc("/jobs", s.logRequest(s.handleJobs))
	mux.HandleFunc("/", s.logRequest(s.handleRoot))
	return mux
}

func (s *server) logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next(w, r)
		log.Printf("%s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	}
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("hello from mesh-runner"))
}

func (s *server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListJobs(w, r)
	case http.MethodPost:
		s.handleCreateJob(w, r)
	default:
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleListJobs(w http.ResponseWriter, _ *http.Request) {
	jobs := s.store.List()

	summaries := make([]jobSummary, len(jobs))
	for i, job := range jobs {
		summaries[i] = jobSummary{
			ID:     job.ID,
			Status: job.Status,
		}
	}

	response := listJobsResponse{
		Jobs: summaries,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

func (s *server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var payload createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Command == "" {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "command is required", http.StatusBadRequest)
		return
	}

	job := s.store.Create(payload.Command)

	jobJson, _ := json.Marshal(job)

	err := s.natsConn.Publish(internal.RequestNATSSubject, jobJson)
	if err != nil {
		log.Printf("error publishing job: %v", err)
	}

	response := createJobResponse{
		ID:        job.ID,
		Command:   job.Command,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}
