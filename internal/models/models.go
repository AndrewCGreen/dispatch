// Package models defines all data structures shared across Dispatch packages.
// This includes database models, API request/response types, and domain constants.
package models

import (
	"encoding/json"
	"time"
)

// --- Subscriber ---

type SubscriberStatus string

const (
	StatusPending      SubscriberStatus = "pending"
	StatusActive       SubscriberStatus = "active"
	StatusUnsubscribed SubscriberStatus = "unsubscribed"
)

type Subscriber struct {
	ID             string           `json:"id"`
	Site           string           `json:"site"`
	Email          string           `json:"email"`
	Name           string           `json:"name,omitempty"`
	Status         SubscriberStatus `json:"status"`
	Attributes     json.RawMessage  `json:"attributes,omitempty"`
	Lists          []string         `json:"lists,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	ConfirmedAt    *time.Time       `json:"confirmed_at,omitempty"`
	UnsubscribedAt *time.Time       `json:"unsubscribed_at,omitempty"`
}

// --- Message ---

type MessageStatus string

const (
	MsgQueued    MessageStatus = "queued"
	MsgSending   MessageStatus = "sending"
	MsgDelivered MessageStatus = "delivered"
	MsgBounced   MessageStatus = "bounced"
	MsgFailed    MessageStatus = "failed"
	MsgSuppressed MessageStatus = "suppressed"
)

type Message struct {
	ID          string          `json:"id"`
	Site        string          `json:"site"`
	ToEmail     string          `json:"to"`
	Template    string          `json:"template,omitempty"`
	Subject     string          `json:"subject,omitempty"`
	Status      MessageStatus   `json:"status"`
	Backend     string          `json:"backend,omitempty"`
	BackendID   string          `json:"backend_id,omitempty"`
	Tags        json.RawMessage `json:"tags,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Error       string          `json:"error,omitempty"`
	QueuedAt    time.Time       `json:"queued_at"`
	SentAt      *time.Time      `json:"sent_at,omitempty"`
	DeliveredAt *time.Time      `json:"delivered_at,omitempty"`
	BouncedAt   *time.Time      `json:"bounced_at,omitempty"`
	FailedAt    *time.Time      `json:"failed_at,omitempty"`
}

// --- Suppression ---

type SuppressionReason string

const (
	ReasonUnsubscribe SuppressionReason = "unsubscribe"
	ReasonBounce      SuppressionReason = "bounce"
	ReasonComplaint   SuppressionReason = "complaint"
	ReasonManual      SuppressionReason = "manual"
	ReasonGDPR        SuppressionReason = "gdpr"
)

type Suppression struct {
	Email     string            `json:"email"`
	Reason    SuppressionReason `json:"reason"`
	Note      string            `json:"note,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// --- Consent ---

type ConsentAction string

const (
	ConsentSubscribe   ConsentAction = "subscribe"
	ConsentUnsubscribe ConsentAction = "unsubscribe"
	ConsentExport      ConsentAction = "export"
	ConsentForget      ConsentAction = "forget"
)

type ConsentRecord struct {
	ID        string        `json:"id"`
	Email     string        `json:"email"`
	Site      string        `json:"site"`
	Action    ConsentAction `json:"action"`
	Source    string        `json:"source,omitempty"`
	IP        string        `json:"ip,omitempty"`
	URL       string        `json:"url,omitempty"`
	Note      string        `json:"note,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// --- Webhook ---

type Webhook struct {
	ID        string    `json:"id"`
	Site      string    `json:"site"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// --- API Key ---

type APIKeyType string

const (
	KeyMaster   APIKeyType = "master"
	KeySite     APIKeyType = "site"
	KeyReadOnly APIKeyType = "readonly"
)

type APIKey struct {
	ID        string     `json:"id"`
	KeyPrefix string     `json:"key_prefix"`
	Type      APIKeyType `json:"type"`
	Site      string     `json:"site,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// --- API Request/Response Types ---

type SendRequest struct {
	To       string            `json:"to"`
	Template string            `json:"template"`
	Data     map[string]any    `json:"data,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SendRawRequest struct {
	To      string            `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html"`
	Text    string            `json:"text,omitempty"`
	Data    map[string]any    `json:"data,omitempty"`
	Tags    []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type BatchSendRequest struct {
	Template   string          `json:"template"`
	Recipients []BatchRecipient `json:"recipients"`
	Tags       []string        `json:"tags,omitempty"`
}

type BatchRecipient struct {
	To   string         `json:"to"`
	Data map[string]any `json:"data,omitempty"`
}

type SendResponse struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	To       string    `json:"to"`
	Template string    `json:"template,omitempty"`
	Site     string    `json:"site"`
	QueuedAt time.Time `json:"queued_at"`
}

type BatchSendResponse struct {
	BatchID    string `json:"batch_id"`
	Total      int    `json:"total"`
	Queued     int    `json:"queued"`
	Suppressed int    `json:"suppressed"`
}

type SubscribeRequest struct {
	Email      string         `json:"email"`
	Name       string         `json:"name,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Lists      []string       `json:"lists,omitempty"`
	Consent    *ConsentInfo   `json:"consent,omitempty"`
}

type ConsentInfo struct {
	Source string `json:"source,omitempty"`
	IP     string `json:"ip,omitempty"`
	URL    string `json:"url,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type PaginatedResponse struct {
	Data    any  `json:"data"`
	Total   int  `json:"total"`
	Page    int  `json:"page"`
	PerPage int  `json:"per_page"`
	HasMore bool `json:"has_more"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Uptime   string            `json:"uptime"`
	Database string            `json:"database"`
	Queue    QueueHealth       `json:"queue"`
	Backends map[string]string `json:"backends"`
}

type QueueHealth struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Failed     int `json:"failed"`
}
