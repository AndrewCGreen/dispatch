// Package token implements HMAC-SHA256 signed tokens for unsubscribe and
// double opt-in confirmation links. Tokens are self-contained: the claims
// are base64url-encoded JSON, signed with a shared secret.
//
// Token format: base64url(JSON_payload) + "." + base64url(HMAC-SHA256_signature)
//
// Claims encode the token type, recipient email, site slug, and an optional
// expiry timestamp. Unsubscribe tokens have no expiry; confirmation tokens
// expire after a configurable TTL (typically 72h).
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TypeUnsubscribe = "unsub"
	TypeConfirm     = "confirm"
)

// Claims holds the data encoded in a token.
type Claims struct {
	Type  string `json:"t"`
	Email string `json:"e"`
	Site  string `json:"s"`
	Exp   int64  `json:"exp,omitempty"` // Unix timestamp; 0 means no expiry
}

// Manager generates and verifies signed tokens.
type Manager struct {
	secret []byte
}

// New creates a Manager using the given secret for HMAC signing.
func New(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Generate creates a signed token for the given type, email, and site.
// Pass ttl=0 for tokens that never expire (e.g. unsubscribe links).
func (m *Manager) Generate(tokenType, email, site string, ttl time.Duration) (string, error) {
	claims := Claims{
		Type:  tokenType,
		Email: email,
		Site:  site,
	}
	if ttl > 0 {
		claims.Exp = time.Now().Add(ttl).Unix()
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := m.sign(encoded)
	return encoded + "." + sig, nil
}

// Verify decodes and validates a token, returning its claims on success.
// Returns an error if the signature is invalid or the token has expired.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	dotIdx := strings.LastIndex(tokenStr, ".")
	if dotIdx < 0 {
		return nil, errors.New("invalid token format")
	}

	encoded := tokenStr[:dotIdx]
	sig := tokenStr[dotIdx+1:]

	expected := m.sign(encoded)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, errors.New("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func (m *Manager) sign(data string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
