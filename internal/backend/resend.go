package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func init() {
	Register("resend", NewResend)
}

// Resend implements the Backend interface for the Resend API.
type Resend struct {
	apiKey string
	client *http.Client
}

// NewResend creates a new Resend backend.
func NewResend(cfg map[string]string) (Backend, error) {
	key := cfg["api_key"]
	if key == "" {
		return nil, fmt.Errorf("resend backend requires api_key")
	}
	return &Resend{
		apiKey: key,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (r *Resend) Name() string { return "resend" }

type resendRequest struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html"`
	Text    string            `json:"text,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Tags    []resendTag       `json:"tags,omitempty"`
}

type resendTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type resendResponse struct {
	ID string `json:"id"`
}

type resendError struct {
	StatusCode int    `json:"statusCode"`
	Name       string `json:"name"`
	Message    string `json:"message"`
}

func (r *Resend) Send(ctx context.Context, msg *OutgoingMessage) (*SendResult, error) {
	from := msg.From
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
	}

	reqBody := resendRequest{
		From:    from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
		Headers: msg.Headers,
	}

	for _, tag := range msg.Tags {
		reqBody.Tags = append(reqBody.Tags, resendTag{Name: "tag", Value: tag})
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr resendError
		json.Unmarshal(respBody, &apiErr)
		return nil, fmt.Errorf("resend API error %d: %s", resp.StatusCode, apiErr.Message)
	}

	var result resendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &SendResult{
		BackendID: result.ID,
		Status:    "sent",
	}, nil
}

func (r *Resend) BatchSend(ctx context.Context, msgs []*OutgoingMessage) (*BatchResult, error) {
	// Resend has a batch endpoint, but for now we do sequential
	result := &BatchResult{}
	for i, msg := range msgs {
		if _, err := r.Send(ctx, msg); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{Index: i, Email: msg.To, Error: err})
		} else {
			result.Succeeded++
		}
	}
	return result, nil
}

func (r *Resend) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.resend.com/domains", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("resend API returned %d", resp.StatusCode)
	}
	return nil
}

func (r *Resend) MaxBatchSize() int { return 100 }
