package sigil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig("secret", "my-svc")
	if c.Secret != "secret" {
		t.Errorf("Secret = %v, want secret", c.Secret)
	}
	if c.ServiceID != "my-svc" {
		t.Errorf("ServiceID = %v, want my-svc", c.ServiceID)
	}
	if c.TimestampTolerance != 30*time.Second {
		t.Errorf("TimestampTolerance = %v, want 30s", c.TimestampTolerance)
	}
}

func TestSigner(t *testing.T) {
	c := Config{
		Secret:             "my-secret",
		ServiceID:          "test-svc",
		TimestampTolerance: 10 * time.Second,
	}
	s := NewSigner(c)

	if got := s.GetServiceID(); got != "test-svc" {
		t.Errorf("GetServiceID() = %v, want test-svc", got)
	}

	body := []byte("hello world")
	headers := s.SignRequest(body)

	if headers[HeaderServiceID] != "test-svc" {
		t.Errorf("HeaderServiceID = %v, want test-svc", headers[HeaderServiceID])
	}
	if headers[HeaderTimestamp] == "" {
		t.Errorf("HeaderTimestamp is empty")
	}
	if headers[HeaderSignature] == "" {
		t.Errorf("HeaderSignature is empty")
	}

	ts, _ := strconv.ParseInt(headers[HeaderTimestamp], 10, 64)
	expectedSig := computeExpectedSignature("my-secret", body, ts)
	if headers[HeaderSignature] != expectedSig {
		t.Errorf("HeaderSignature = %v, want %v", headers[HeaderSignature], expectedSig)
	}
}

func computeExpectedSignature(secret string, body []byte, timestamp int64) string {
	message := fmt.Sprintf("%d:%s", timestamp, string(body))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func TestVerifier_VerifyRequest(t *testing.T) {
	c := Config{
		Secret:             "my-secret",
		ServiceID:          "target-svc",
		TimestampTolerance: 5 * time.Minute,
	}
	v := NewVerifier(c, []string{"allowed-svc"})
	body := []byte("test-data")
	now := time.Now().Unix()

	validSig := computeExpectedSignature("my-secret", body, now)
	expiredSig := computeExpectedSignature("my-secret", body, now-600) // 10 mins ago

	tests := []struct {
		name         string
		serviceID    string
		timestampStr string
		signature    string
		body         []byte
		wantErr      error
	}{
		{
			name:         "valid request",
			serviceID:    "allowed-svc",
			timestampStr: strconv.FormatInt(now, 10),
			signature:    validSig,
			body:         body,
			wantErr:      nil,
		},
		{
			name:         "missing service ID",
			serviceID:    "",
			timestampStr: strconv.FormatInt(now, 10),
			signature:    validSig,
			body:         body,
			wantErr:      ErrMissingHeaders,
		},
		{
			name:         "missing timestamp",
			serviceID:    "allowed-svc",
			timestampStr: "",
			signature:    validSig,
			body:         body,
			wantErr:      ErrMissingHeaders,
		},
		{
			name:         "missing signature",
			serviceID:    "allowed-svc",
			timestampStr: strconv.FormatInt(now, 10),
			signature:    "",
			body:         body,
			wantErr:      ErrMissingHeaders,
		},
		{
			name:         "unknown service",
			serviceID:    "unknown-svc",
			timestampStr: strconv.FormatInt(now, 10),
			signature:    validSig,
			body:         body,
			wantErr:      ErrUnknownService,
		},
		{
			name:         "invalid timestamp format",
			serviceID:    "allowed-svc",
			timestampStr: "not-a-number",
			signature:    validSig,
			body:         body,
			wantErr:      ErrInvalidTimestamp,
		},
		{
			name:         "timestamp expired",
			serviceID:    "allowed-svc",
			timestampStr: strconv.FormatInt(now-600, 10), // 10 mins ago
			signature:    expiredSig,
			body:         body,
			wantErr:      ErrTimestampExpired,
		},
		{
			name:         "invalid signature",
			serviceID:    "allowed-svc",
			timestampStr: strconv.FormatInt(now, 10),
			signature:    "wrong-sig",
			body:         body,
			wantErr:      ErrInvalidSignature,
		},
		{
			name:         "future timestamp",
			serviceID:    "allowed-svc",
			timestampStr: strconv.FormatInt(now+100, 10), // in the future
			signature:    computeExpectedSignature("my-secret", body, now+100),
			body:         body,
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.VerifyRequest(tt.serviceID, tt.timestampStr, tt.signature, tt.body)
			if err != tt.wantErr {
				t.Errorf("VerifyRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
