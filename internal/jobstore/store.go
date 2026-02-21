package jobstore

import (
	"sync"
	"time"

	"github.com/zaminda/mesh-runner/internal"
)

type Store struct {
	mu     sync.Mutex
	jobs   []internal.Job
	nextID int
}

func New() *Store {
	return &Store{
		nextID: 1000,
	}
}

func (s *Store) Create(command string) internal.Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := internal.Job{
		ID:        s.nextID,
		Command:   command,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.jobs = append(s.jobs, job)
	return job
}

func (s *Store) List() []internal.Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.Job, len(s.jobs))
	copy(result, s.jobs)
	return result
}
