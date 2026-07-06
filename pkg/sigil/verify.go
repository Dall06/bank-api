package sigil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	// ErrMissingHeaders is returned when required headers are missing
	ErrMissingHeaders = errors.New("missing required sigil headers")
	// ErrInvalidTimestamp is returned when the timestamp is invalid
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	// ErrTimestampExpired is returned when the timestamp is too old
	ErrTimestampExpired = errors.New("request timestamp expired")
	// ErrInvalidSignature is returned when the signature doesn't match
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrUnknownService is returned when the service ID is not in the whitelist
	ErrUnknownService = errors.New("unknown service ID")
)

// Verifier validates HMAC signatures on incoming requests
type Verifier struct {
	config          Config
	allowedServices map[string]bool
}

// NewVerifier creates a new Verifier
// allowedServices is a list of service IDs that are allowed to make requests
func NewVerifier(config Config, allowedServices []string) *Verifier {
	allowed := make(map[string]bool, len(allowedServices))
	for _, svc := range allowedServices {
		allowed[svc] = true
	}

	return &Verifier{
		config:          config,
		allowedServices: allowed,
	}
}

// VerifyRequest validates the signature headers on an incoming request
func (v *Verifier) VerifyRequest(serviceID, timestampStr, signature string, body []byte) error {
	// Check required headers
	if serviceID == "" || timestampStr == "" || signature == "" {
		return ErrMissingHeaders
	}

	// Verify service is allowed
	if !v.allowedServices[serviceID] {
		return ErrUnknownService
	}

	// Parse timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}

	// Check timestamp is within tolerance
	requestTime := time.Unix(timestamp, 0)
	now := time.Now()
	diff := now.Sub(requestTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > v.config.TimestampTolerance {
		return ErrTimestampExpired
	}

	// Verify signature
	expectedSig := v.computeSignature(body, timestamp)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return ErrInvalidSignature
	}

	return nil
}

// computeSignature generates the expected HMAC-SHA256 signature
func (v *Verifier) computeSignature(body []byte, timestamp int64) string {
	// Create the message to sign: timestamp + body
	message := fmt.Sprintf("%d:%s", timestamp, string(body))

	h := hmac.New(sha256.New, []byte(v.config.Secret))
	h.Write([]byte(message))

	return hex.EncodeToString(h.Sum(nil))
}
