// Package store implements all database operations for Dispatch.
// It supports SQLite (default) and PostgreSQL. Migrations are embedded
// and applied automatically on startup via Migrate().
//
// All public methods accept a context.Context for cancellation support.
// Database-level locking is used for queue operations to prevent duplicate sends.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
	"github.com/dispatch-email/dispatch/internal/models"
)

// Store handles all database operations.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given driver and DSN.
func New(driver, dsn string) (*Store, error) {
	if driver == "" || driver == "sqlite" {
		driver = "sqlite3"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// SQLite optimizations
	if driver == "sqlite3" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=5000")
		db.Exec("PRAGMA synchronous=NORMAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate runs database migrations.
func (s *Store) Migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS subscribers (
    id              TEXT PRIMARY KEY,
    site            TEXT NOT NULL,
    email           TEXT NOT NULL,
    name            TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    attributes      TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at    TIMESTAMP,
    unsubscribed_at TIMESTAMP,
    UNIQUE(site, email)
);

CREATE TABLE IF NOT EXISTS subscriber_lists (
    subscriber_id TEXT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    list_slug     TEXT NOT NULL,
    subscribed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (subscriber_id, list_slug)
);

CREATE TABLE IF NOT EXISTS consent_log (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    site        TEXT NOT NULL,
    action      TEXT NOT NULL,
    source      TEXT,
    ip          TEXT,
    url         TEXT,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS suppressions (
    email       TEXT PRIMARY KEY,
    reason      TEXT NOT NULL,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id           TEXT PRIMARY KEY,
    site         TEXT NOT NULL,
    to_email     TEXT NOT NULL,
    template     TEXT,
    subject      TEXT,
    status       TEXT NOT NULL DEFAULT 'queued',
    backend      TEXT,
    backend_id   TEXT,
    tags         TEXT,
    metadata     TEXT,
    error        TEXT,
    queued_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at      TIMESTAMP,
    delivered_at TIMESTAMP,
    bounced_at   TIMESTAMP,
    failed_at    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS send_queue (
    id          TEXT PRIMARY KEY,
    message_id  TEXT NOT NULL REFERENCES messages(id),
    site        TEXT NOT NULL,
    attempts    INTEGER NOT NULL DEFAULT 0,
    next_retry  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_by   TEXT,
    locked_at   TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT PRIMARY KEY,
    site        TEXT NOT NULL,
    url         TEXT NOT NULL,
    events      TEXT NOT NULL,
    secret      TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          TEXT PRIMARY KEY,
    key_hash    TEXT NOT NULL UNIQUE,
    key_prefix  TEXT NOT NULL,
    type        TEXT NOT NULL,
    site        TEXT,
    name        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used   TIMESTAMP,
    revoked_at  TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscribers_site_email ON subscribers(site, email);
CREATE INDEX IF NOT EXISTS idx_messages_site ON messages(site, queued_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_send_queue_next ON send_queue(next_retry);
CREATE INDEX IF NOT EXISTS idx_suppressions_email ON suppressions(email);
`

// --- Suppression Operations ---

// IsSuppressed checks if an email is on the global suppression list.
func (s *Store) IsSuppressed(ctx context.Context, email string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM suppressions WHERE email = ?", email).Scan(&count)
	return count > 0, err
}

// AddSuppression adds an email to the suppression list.
func (s *Store) AddSuppression(ctx context.Context, sup *models.Suppression) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO suppressions (email, reason, note, created_at) VALUES (?, ?, ?, ?)",
		sup.Email, sup.Reason, sup.Note, time.Now().UTC(),
	)
	return err
}

// RemoveSuppression removes an email from the suppression list.
func (s *Store) RemoveSuppression(ctx context.Context, email string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM suppressions WHERE email = ?", email)
	return err
}

// ListSuppressions returns all suppressed emails.
func (s *Store) ListSuppressions(ctx context.Context, page, perPage int) ([]models.Suppression, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM suppressions").Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx,
		"SELECT email, reason, note, created_at FROM suppressions ORDER BY created_at DESC LIMIT ? OFFSET ?",
		perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sups []models.Suppression
	for rows.Next() {
		var sup models.Suppression
		if err := rows.Scan(&sup.Email, &sup.Reason, &sup.Note, &sup.CreatedAt); err != nil {
			return nil, 0, err
		}
		sups = append(sups, sup)
	}
	return sups, total, nil
}

// --- Subscriber Operations ---

// CreateSubscriber inserts a new subscriber.
func (s *Store) CreateSubscriber(ctx context.Context, sub *models.Subscriber) error {
	sub.ID = uuid.New().String()
	sub.CreatedAt = time.Now().UTC()

	attrs, _ := json.Marshal(sub.Attributes)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subscribers (id, site, email, name, status, attributes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sub.ID, sub.Site, sub.Email, sub.Name, sub.Status, string(attrs), sub.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating subscriber: %w", err)
	}

	// Add list memberships
	for _, list := range sub.Lists {
		_, err := s.db.ExecContext(ctx,
			"INSERT OR IGNORE INTO subscriber_lists (subscriber_id, list_slug) VALUES (?, ?)",
			sub.ID, list,
		)
		if err != nil {
			return fmt.Errorf("adding list membership: %w", err)
		}
	}

	return nil
}

// GetSubscriber retrieves a subscriber by site and email.
func (s *Store) GetSubscriber(ctx context.Context, site, email string) (*models.Subscriber, error) {
	var sub models.Subscriber
	var attrs sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, site, email, name, status, attributes, created_at, confirmed_at, unsubscribed_at
		 FROM subscribers WHERE site = ? AND email = ?`,
		site, email,
	).Scan(&sub.ID, &sub.Site, &sub.Email, &sub.Name, &sub.Status, &attrs,
		&sub.CreatedAt, &sub.ConfirmedAt, &sub.UnsubscribedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if attrs.Valid {
		sub.Attributes = json.RawMessage(attrs.String)
	}

	// Load lists
	rows, err := s.db.QueryContext(ctx,
		"SELECT list_slug FROM subscriber_lists WHERE subscriber_id = ?", sub.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var list string
		if err := rows.Scan(&list); err != nil {
			return nil, err
		}
		sub.Lists = append(sub.Lists, list)
	}

	return &sub, nil
}

// DeleteSubscriber removes a subscriber and their list memberships.
func (s *Store) DeleteSubscriber(ctx context.Context, site, email string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM subscribers WHERE site = ? AND email = ?", site, email)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("subscriber not found")
	}
	return nil
}

// --- Message Operations ---

// CreateMessage inserts a new message into the messages table and send queue.
func (s *Store) CreateMessage(ctx context.Context, msg *models.Message) error {
	msg.ID = "msg_" + uuid.New().String()[:12]
	msg.QueuedAt = time.Now().UTC()
	msg.Status = models.MsgQueued

	tags, _ := json.Marshal(msg.Tags)
	meta, _ := json.Marshal(msg.Metadata)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO messages (id, site, to_email, template, subject, status, backend, tags, metadata, queued_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.Site, msg.ToEmail, msg.Template, msg.Subject, msg.Status, msg.Backend,
		string(tags), string(meta), msg.QueuedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting message: %w", err)
	}

	queueID := "q_" + uuid.New().String()[:12]
	_, err = tx.ExecContext(ctx,
		`INSERT INTO send_queue (id, message_id, site, created_at) VALUES (?, ?, ?, ?)`,
		queueID, msg.ID, msg.Site, msg.QueuedAt,
	)
	if err != nil {
		return fmt.Errorf("enqueuing message: %w", err)
	}

	return tx.Commit()
}

// GetMessage retrieves a message by ID.
func (s *Store) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	var msg models.Message
	var tags, meta sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, site, to_email, template, subject, status, backend, backend_id,
		        tags, metadata, error, queued_at, sent_at, delivered_at, bounced_at, failed_at
		 FROM messages WHERE id = ?`, id,
	).Scan(&msg.ID, &msg.Site, &msg.ToEmail, &msg.Template, &msg.Subject, &msg.Status,
		&msg.Backend, &msg.BackendID, &tags, &meta, &msg.Error,
		&msg.QueuedAt, &msg.SentAt, &msg.DeliveredAt, &msg.BouncedAt, &msg.FailedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if tags.Valid {
		msg.Tags = json.RawMessage(tags.String)
	}
	if meta.Valid {
		msg.Metadata = json.RawMessage(meta.String)
	}
	return &msg, nil
}

// --- Consent Operations ---

// LogConsent records a consent event.
func (s *Store) LogConsent(ctx context.Context, rec *models.ConsentRecord) error {
	rec.ID = uuid.New().String()
	rec.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO consent_log (id, email, site, action, source, ip, url, note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.Email, rec.Site, rec.Action, rec.Source, rec.IP, rec.URL, rec.Note, rec.CreatedAt,
	)
	return err
}

// --- Queue Operations ---

type QueueItem struct {
	ID        string
	MessageID string
	Site      string
	Attempts  int
}

// DequeueMessages claims up to n messages from the send queue.
func (s *Store) DequeueMessages(ctx context.Context, workerID string, n int) ([]QueueItem, error) {
	now := time.Now().UTC()
	lockExpiry := now.Add(-5 * time.Minute) // stale locks

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, message_id, site, attempts FROM send_queue
		 WHERE (locked_by IS NULL OR locked_at < ?) AND next_retry <= ?
		 ORDER BY next_retry ASC LIMIT ?`,
		lockExpiry, now, n,
	)
	if err != nil {
		return nil, err
	}

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		if err := rows.Scan(&item.ID, &item.MessageID, &item.Site, &item.Attempts); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	rows.Close()

	for _, item := range items {
		_, err := tx.ExecContext(ctx,
			"UPDATE send_queue SET locked_by = ?, locked_at = ? WHERE id = ?",
			workerID, now, item.ID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

// CompleteQueueItem removes a successfully sent item from the queue.
func (s *Store) CompleteQueueItem(ctx context.Context, queueID, backendID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the message ID
	var msgID string
	err = tx.QueryRowContext(ctx, "SELECT message_id FROM send_queue WHERE id = ?", queueID).Scan(&msgID)
	if err != nil {
		return err
	}

	// Update message status
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx,
		"UPDATE messages SET status = ?, backend_id = ?, sent_at = ? WHERE id = ?",
		models.MsgDelivered, backendID, now, msgID,
	)
	if err != nil {
		return err
	}

	// Remove from queue
	_, err = tx.ExecContext(ctx, "DELETE FROM send_queue WHERE id = ?", queueID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// FailQueueItem marks a queue item as failed and schedules a retry or gives up.
func (s *Store) FailQueueItem(ctx context.Context, queueID string, sendErr error, maxRetries int, backoff []time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var msgID string
	var attempts int
	err = tx.QueryRowContext(ctx,
		"SELECT message_id, attempts FROM send_queue WHERE id = ?", queueID,
	).Scan(&msgID, &attempts)
	if err != nil {
		return err
	}

	attempts++

	if attempts >= maxRetries {
		// Give up
		now := time.Now().UTC()
		_, err = tx.ExecContext(ctx,
			"UPDATE messages SET status = ?, error = ?, failed_at = ? WHERE id = ?",
			models.MsgFailed, sendErr.Error(), now, msgID,
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM send_queue WHERE id = ?", queueID)
		if err != nil {
			return err
		}
	} else {
		// Retry
		backoffIdx := attempts - 1
		if backoffIdx >= len(backoff) {
			backoffIdx = len(backoff) - 1
		}
		nextRetry := time.Now().UTC().Add(backoff[backoffIdx])

		_, err = tx.ExecContext(ctx,
			"UPDATE send_queue SET attempts = ?, next_retry = ?, locked_by = NULL, locked_at = NULL WHERE id = ?",
			attempts, nextRetry, queueID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// QueueStats returns counts of queue items by status.
func (s *Store) QueueStats(ctx context.Context) (pending, processing, failed int, err error) {
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM send_queue WHERE locked_by IS NULL").Scan(&pending)
	if err != nil {
		return
	}
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM send_queue WHERE locked_by IS NOT NULL").Scan(&processing)
	if err != nil {
		return
	}
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM messages WHERE status = 'failed'").Scan(&failed)
	return
}
