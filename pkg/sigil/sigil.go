// Package sigil provides HMAC-SHA256 request signing for service-to-service authentication.
// It allows the gateway to sign outgoing requests and backend services to verify them.
package sigil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const (
	// HeaderServiceID identifies the calling service
	HeaderServiceID = "X-Service-ID"
	// HeaderTimestamp contains the request timestamp (Unix seconds)
	HeaderTimestamp = "X-Service-Timestamp"
	// HeaderSignature contains the HMAC-SHA256 signature
	HeaderSignature = "X-Service-Signature"
)

// Config holds sigil configuration
type Config struct {
	// Secret is the shared HMAC secret key
	Secret string
	// ServiceID is the identifier for this service
	ServiceID string
	// TimestampTolerance is the maximum allowed time difference for requests
	TimestampTolerance time.Duration
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig(secret, serviceID string) Config {
	return Config{
		Secret:             secret,
		ServiceID:          serviceID,
		TimestampTolerance: 30 * time.Second,
	}
}

// Signer creates HMAC signatures for outgoing requests
type Signer struct {
	config Config
}

// NewSigner creates a new Signer
func NewSigner(config Config) *Signer {
	return &Signer{config: config}
}

// SignRequest generates signature headers for a request
// Returns headers: X-Service-ID, X-Service-Timestamp, X-Service-Signature
func (s *Signer) SignRequest(body []byte) map[string]string {
	timestamp := time.Now().Unix()
	signature := s.computeSignature(body, timestamp)

	return map[string]string{
		HeaderServiceID: s.config.ServiceID,
		HeaderTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignature: signature,
	}
}

// computeSignature generates the HMAC-SHA256 signature
func (s *Signer) computeSignature(body []byte, timestamp int64) string {
	// Create the message to sign: timestamp + body
	message := fmt.Sprintf("%d:%s", timestamp, string(body))

	h := hmac.New(sha256.New, []byte(s.config.Secret))
	h.Write([]byte(message))

	return hex.EncodeToString(h.Sum(nil))
}

// GetServiceID returns the service ID
func (s *Signer) GetServiceID() string {
	return s.config.ServiceID
}
