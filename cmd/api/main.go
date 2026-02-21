package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zaminda/mesh-runner/internal"
)

type createJobRequest struct {
	Command string `json:"command"`
}

func main() {
	mux := http.NewServeMux()
	var (
		jobs []internal.Job
		mu   sync.Mutex
	)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload createJobRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if payload.Command == "" {
			http.Error(w, "command is required", http.StatusBadRequest)
			return
		}

		newJob := internal.Job{
			ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
			Command:   payload.Command,
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
		}

		mu.Lock()
		jobs = append(jobs, newJob)
		mu.Unlock()

		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from mesh-runner"))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
