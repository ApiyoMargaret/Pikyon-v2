package auth

import (
	"context"
	"log"
	"runtime/debug"
	"time"
)

type Worker struct {
	service  Service
	interval time.Duration
}

func NewWorker(service Service, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 1 * time.Hour // Default cleanup interval
	}
	return &Worker{
		service:  service,
		interval: interval,
	}
}

// Start runs the periodic background cleanup loop until parentCtx is canceled.
func (w *Worker) Start(parentCtx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("[SessionWorker] Started background session cleanup worker (interval: %s)", w.interval)

	for {
		select {
		case <-parentCtx.Done():
			log.Println("[SessionWorker] Stopping background session cleanup worker...")
			return
		case <-ticker.C:
			w.runCleanupJob(parentCtx)
		}
	}
}

func (w *Worker) runCleanupJob(parentCtx context.Context) {
	// 1. Recovery Protection Against Internal Query Panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SessionWorker] PANIC RECOVERED in background cleanup worker: %v\nStack trace:\n%s", r, string(debug.Stack()))
		}
	}()

	// 2. Short-Lived Isolated Context to Prevent Parent Context Expiry Lockout
	jobCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Short-circuit execution if application shutdown was signaled prior to starting job
	if parentCtx.Err() != nil {
		return
	}

	deleted, err := w.service.CleanExpiredSessions(jobCtx)
	if err != nil {
		log.Printf("[SessionWorker] Error cleaning expired sessions: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("[SessionWorker] Successfully cleaned up %d expired session(s)", deleted)
	}
}