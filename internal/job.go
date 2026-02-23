package internal

import "time"

const (
	Pending = "PENDING"
	Done    = "DONE"
	Failed  = "FAILED"
	Queued  = "QUEUED"

	RequestNATSSubject = "mesh.job.request"
	ResultNATSSubject  = "mesh.job.request"
)

type Job struct {
	ID        int       `json:"id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
