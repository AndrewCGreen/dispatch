// Package queue implements a reliable, database-backed send queue for Dispatch.
//
// The queue uses database-level locking (locked_by, locked_at columns) to
// allow multiple workers to process items concurrently without duplicates.
// Stale locks (older than 5 minutes) are automatically released, providing
// crash recovery without manual intervention.
//
// Workers poll the send_queue table every second for unlocked items where
// next_retry <= now. On failure, items are rescheduled with exponential
// backoff configured via Options.RetryBackoff. After Options.RetryMax
// attempts, items are removed from the queue and the message is marked failed.
package queue

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dispatch-email/dispatch/internal/backend"
	"github.com/dispatch-email/dispatch/internal/store"
)

// Options configures the send queue.
type Options struct {
	Workers      int
	RetryMax     int
	RetryBackoff string // comma-separated durations: "5m,30m,2h"
}

// Queue processes the send queue with multiple workers.
type Queue struct {
	store    *store.Store
	backends *backend.Router
	opts     Options
	backoff  []time.Duration
	wg       sync.WaitGroup
}

// New creates a new Queue.
func New(store *store.Store, backends *backend.Router, opts Options) *Queue {
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	if opts.RetryMax <= 0 {
		opts.RetryMax = 3
	}

	backoff := parseBackoff(opts.RetryBackoff)

	return &Queue{
		store:    store,
		backends: backends,
		opts:     opts,
		backoff:  backoff,
	}
}

// Start launches queue worker goroutines.
func (q *Queue) Start(ctx context.Context) {
	for i := 0; i < q.opts.Workers; i++ {
		q.wg.Add(1)
		workerID := fmt.Sprintf("worker-%d", i)
		go q.worker(ctx, workerID)
	}
	slog.Info("queue started", "workers", q.opts.Workers)
}

// Wait blocks until all workers have stopped.
func (q *Queue) Wait() {
	q.wg.Wait()
}

func (q *Queue) worker(ctx context.Context, id string) {
	defer q.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("queue worker stopping", "worker", id)
			return
		case <-ticker.C:
			q.processItems(ctx, id)
		}
	}
}

func (q *Queue) processItems(ctx context.Context, workerID string) {
	items, err := q.store.DequeueMessages(ctx, workerID, 10)
	if err != nil {
		slog.Error("dequeue failed", "worker", workerID, "error", err)
		return
	}

	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := q.processOne(ctx, item); err != nil {
			slog.Error("send failed",
				"worker", workerID,
				"message_id", item.MessageID,
				"site", item.Site,
				"attempt", item.Attempts+1,
				"error", err,
			)
			q.store.FailQueueItem(ctx, item.ID, err, q.opts.RetryMax, q.backoff)
		}
	}
}

func (q *Queue) processOne(ctx context.Context, item store.QueueItem) error {
	// Load the message
	msg, err := q.store.GetMessage(ctx, item.MessageID)
	if err != nil {
		return fmt.Errorf("loading message: %w", err)
	}
	if msg == nil {
		return fmt.Errorf("message %s not found", item.MessageID)
	}

	// Check suppression
	suppressed, err := q.store.IsSuppressed(ctx, msg.ToEmail)
	if err != nil {
		return fmt.Errorf("checking suppression: %w", err)
	}
	if suppressed {
		slog.Info("skipping suppressed recipient", "email", msg.ToEmail, "message_id", msg.ID)
		q.store.CompleteQueueItem(ctx, item.ID, "")
		return nil
	}

	// Get the backend for this site
	b, err := q.backends.ForSite(item.Site)
	if err != nil {
		return fmt.Errorf("getting backend: %w", err)
	}

	// Build the outgoing message from the stored rendered content.
	outgoing := &backend.OutgoingMessage{
		MessageID: msg.ID,
		From:      msg.FromEmail,
		FromName:  msg.FromName,
		To:        msg.ToEmail,
		Subject:   msg.Subject,
		HTML:      msg.HTMLBody,
		Text:      msg.TextBody,
	}

	if msg.ListUnsubscribe != "" {
		outgoing.Headers = map[string]string{
			"List-Unsubscribe":      "<" + msg.ListUnsubscribe + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		}
	}

	result, err := b.Send(ctx, outgoing)
	if err != nil {
		return err
	}

	// Mark as complete
	if err := q.store.CompleteQueueItem(ctx, item.ID, result.BackendID); err != nil {
		slog.Error("failed to complete queue item", "queue_id", item.ID, "error", err)
	}

	slog.Info("email sent",
		"message_id", msg.ID,
		"to", msg.ToEmail,
		"backend", b.Name(),
		"backend_id", result.BackendID,
	)

	return nil
}

func parseBackoff(s string) []time.Duration {
	if s == "" {
		return []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour}
	}

	var durations []time.Duration
	for _, part := range strings.Split(s, ",") {
		d, err := time.ParseDuration(strings.TrimSpace(part))
		if err != nil {
			slog.Warn("invalid backoff duration, using default", "value", part)
			d = 5 * time.Minute
		}
		durations = append(durations, d)
	}
	return durations
}
