package requestor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_Do_Single(t *testing.T) {
	tests := []struct {
		name       string
		roundTrip  roundTripperFunc
		reqBody    []byte
		wantStatus int
		wantBody   []byte
		wantErr    bool
	}{
		{
			name: "success single request",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("ok")),
				}, nil
			},
			reqBody:    nil,
			wantStatus: 200,
			wantBody:   []byte("ok"),
			wantErr:    false,
		},
		{
			name: "http error",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
			reqBody:    nil,
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
		{
			name: "read error body",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(&errorReader{}),
				}, nil
			},
			reqBody:    nil,
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&http.Client{Transport: tt.roundTrip})
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			if tt.reqBody != nil {
				req.Body = io.NopCloser(bytes.NewBuffer(tt.reqBody))
			}

			resp, err := client.Do(context.Background(), req, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if resp.StatusCode != tt.wantStatus {
					t.Errorf("StatusCode = %v, want %v", resp.StatusCode, tt.wantStatus)
				}
				if string(resp.Bytes) != string(tt.wantBody) {
					t.Errorf("Bytes = %v, want %v", string(resp.Bytes), string(tt.wantBody))
				}
			}
		})
	}
}

func TestClient_Do_Retry(t *testing.T) {
	tests := []struct {
		name       string
		reqBody    []byte
		setup      func() roundTripperFunc
		wantStatus int
		wantBody   []byte
		wantErr    bool
		cancelCtx  bool
	}{
		{
			name:    "success on first attempt",
			reqBody: []byte("payload"),
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString("ok")),
					}, nil
				}
			},
			wantStatus: 200,
			wantBody:   []byte("ok"),
			wantErr:    false,
		},
		{
			name:    "success after retries (500)",
			reqBody: []byte("payload"),
			setup: func() roundTripperFunc {
				attempts := 0
				return func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 3 {
						return &http.Response{
							StatusCode: 500,
							Body:       io.NopCloser(bytes.NewBufferString("error")),
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString("recovered")),
					}, nil
				}
			},
			wantStatus: 200,
			wantBody:   []byte("recovered"),
			wantErr:    false,
		},
		{
			name:    "exhaust retries on 503",
			reqBody: nil,
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 503,
						Body:       io.NopCloser(bytes.NewBufferString("unavailable")),
					}, nil
				}
			},
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
		{
			name:    "exhaust retries on network error",
			reqBody: nil,
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network failure")
				}
			},
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
		{
			name:    "context deadline exceeded",
			reqBody: nil,
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return nil, context.DeadlineExceeded
				}
			},
			cancelCtx:  true, // Cancel the context to cover ctx.Err()
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
		{
			name:    "timeout from http client",
			reqBody: nil,
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return nil, context.DeadlineExceeded
				}
			},
			cancelCtx:  false, // don't cancel ctx, let it loop and return the error
			wantStatus: 0,
			wantBody:   nil,
			wantErr:    true,
		},
		{
			name: "body read error for retry setup",
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return nil, nil
				}
			},
			reqBody:    nil, // Will be set specially below
			wantErr:    true,
		},
		{
			name: "read error on successful response",
			setup: func() roundTripperFunc {
				return func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(&errorReader{}),
					}, nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&http.Client{Transport: tt.setup()})
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

			if tt.name == "body read error for retry setup" {
				req.Body = io.NopCloser(&errorReader{})
			} else if tt.reqBody != nil {
				req.Body = io.NopCloser(bytes.NewBuffer(tt.reqBody))
			}

			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // cancel immediately
			} else {
				// Reduce the wait time for tests
				ctx, _ = context.WithTimeout(ctx, 2*time.Second)
			}

			resp, err := client.Do(ctx, req, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if resp.StatusCode != tt.wantStatus {
					t.Errorf("StatusCode = %v, want %v", resp.StatusCode, tt.wantStatus)
				}
				if string(resp.Bytes) != string(tt.wantBody) {
					t.Errorf("Bytes = %v, want %v", string(resp.Bytes), string(tt.wantBody))
				}
			}
		})
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestClient_Sleep(t *testing.T) {
	client := NewClient(http.DefaultClient)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	
	// Should return immediately without waiting for the 10-second duration
	start := time.Now()
	client.sleep(ctx, 10*time.Second)
	if time.Since(start) > 1*time.Second {
		t.Error("sleep did not exit immediately on cancelled context")
	}
}
