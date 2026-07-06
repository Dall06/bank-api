package middlewares_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAkagi(t *testing.T) {
	tests := []struct {
		name              string
		expectedStatus    int
		validateResponse  func(*testing.T, *httptest.ResponseRecorder)
		validateTimestamp bool
	}{
		{
			name:           "returns 200 OK status",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
			},
			validateTimestamp: true,
		},
		{
			name:           "returns JSON content type",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				contentType := rec.Header().Get("Content-Type")
				assert.Contains(t, contentType, "application/json")
			},
			validateTimestamp: true,
		},
		{
			name:           "returns valid JSON body",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err, "response should be valid JSON")
			},
			validateTimestamp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			handler := middlewares.Akagi()
			err := handler(c)

			// Assertions
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateResponse != nil {
				tt.validateResponse(t, rec)
			}
		})
	}
}

func TestAkagi_ResponseStructure(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus string
		validateFields bool
	}{
		{
			name:           "response contains status field",
			expectedStatus: "OK",
			validateFields: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middlewares.Akagi()
			err := handler(c)
			require.NoError(t, err)

			// Parse response
			var health middlewares.Health
			err = json.Unmarshal(rec.Body.Bytes(), &health)
			require.NoError(t, err)

			if tt.validateFields {
				assert.Equal(t, tt.expectedStatus, health.Status)
				assert.NotZero(t, health.Timestamp, "timestamp should not be zero")
			}
		})
	}
}

func TestAkagi_Timestamp(t *testing.T) {
	tests := []struct {
		name           string
		validateUTC    bool
		validateRecent bool
		maxAge         time.Duration
	}{
		{
			name:           "timestamp is in UTC",
			validateUTC:    true,
			validateRecent: true,
			maxAge:         1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			before := time.Now().UTC()
			handler := middlewares.Akagi()
			err := handler(c)
			after := time.Now().UTC()

			require.NoError(t, err)

			var health middlewares.Health
			err = json.Unmarshal(rec.Body.Bytes(), &health)
			require.NoError(t, err)

			if tt.validateUTC {
				// Verify timestamp is UTC
				assert.Equal(t, "UTC", health.Timestamp.Location().String(),
					"timestamp should be in UTC timezone")
			}

			if tt.validateRecent {
				// Verify timestamp is recent (within the test execution time)
				assert.True(t, health.Timestamp.Equal(before) || health.Timestamp.After(before),
					"timestamp should be after or equal to before time")
				assert.True(t, health.Timestamp.Equal(after) || health.Timestamp.Before(after),
					"timestamp should be before or equal to after time")
			}
		})
	}
}

func TestAkagi_StatusValue(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus string
	}{
		{
			name:           "status is exactly 'OK'",
			expectedStatus: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middlewares.Akagi()
			err := handler(c)
			require.NoError(t, err)

			var health middlewares.Health
			err = json.Unmarshal(rec.Body.Bytes(), &health)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, health.Status)
		})
	}
}

func TestAkagi_MultipleRequests(t *testing.T) {
	// Test that multiple requests work correctly
	handler := middlewares.Akagi()

	for i := 0; i < 10; i++ {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var health middlewares.Health
		err = json.Unmarshal(rec.Body.Bytes(), &health)
		require.NoError(t, err)
		assert.Equal(t, "OK", health.Status)
	}
}

func TestAkagi_ConcurrentRequests(t *testing.T) {
	// Test that the handler is safe for concurrent use
	handler := middlewares.Akagi()

	done := make(chan bool, 100)
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		go func() {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)
			if err != nil {
				errors <- err
			} else {
				var health middlewares.Health
				err = json.Unmarshal(rec.Body.Bytes(), &health)
				if err != nil || health.Status != "OK" {
					errors <- err
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	close(errors)

	// No errors should occur
	errorCount := 0
	for range errors {
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "concurrent requests should all succeed")
}

func TestAkagi_HTTPMethods(t *testing.T) {
	// Test that the handler works with different HTTP methods
	tests := []struct {
		name   string
		method string
	}{
		{"GET request", http.MethodGet},
		{"POST request", http.MethodPost},
		{"PUT request", http.MethodPut},
		{"DELETE request", http.MethodDelete},
		{"PATCH request", http.MethodPatch},
		{"HEAD request", http.MethodHead},
		{"OPTIONS request", http.MethodOptions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middlewares.Akagi()
			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestAkagi_ResponseFormat(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middlewares.Akagi()
	err := handler(c)
	require.NoError(t, err)

	// Verify exact JSON structure
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have exactly 2 fields: status and timestamp
	assert.Len(t, response, 2, "response should have exactly 2 fields")
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")

	// Verify types
	status, ok := response["status"].(string)
	assert.True(t, ok, "status should be a string")
	assert.Equal(t, "OK", status)

	timestamp, ok := response["timestamp"].(string)
	assert.True(t, ok, "timestamp should be a string")
	assert.NotEmpty(t, timestamp)

	// Verify timestamp is valid RFC3339
	_, err = time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err, "timestamp should be valid RFC3339 format")
}

func TestAkagi_TimestampPrecision(t *testing.T) {
	// Test that timestamps from multiple calls are different
	handler := middlewares.Akagi()
	timestamps := make([]time.Time, 0, 10)

	for i := 0; i < 10; i++ {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)

		var health middlewares.Health
		err = json.Unmarshal(rec.Body.Bytes(), &health)
		require.NoError(t, err)

		timestamps = append(timestamps, health.Timestamp)

		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Verify timestamps are in ascending order (or at least not identical)
	allSame := true
	for i := 1; i < len(timestamps); i++ {
		if !timestamps[i].Equal(timestamps[0]) {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "timestamps should vary across requests")
}

func TestAkagi_NoSideEffects(t *testing.T) {
	// Test that the handler doesn't modify the request or have side effects
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	originalURL := req.URL.String()
	originalMethod := req.Method

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middlewares.Akagi()
	err := handler(c)
	require.NoError(t, err)

	// Verify request is unchanged
	assert.Equal(t, originalURL, req.URL.String(), "request URL should not change")
	assert.Equal(t, originalMethod, req.Method, "request method should not change")
}

func TestAkagi_WithContext(t *testing.T) {
	// Test that handler works with context values set
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set some context values
	c.Set("user_id", "123")
	c.Set("trace_id", "abc-def")

	handler := middlewares.Akagi()
	err := handler(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Context values should still be there
	assert.Equal(t, "123", c.Get("user_id"))
	assert.Equal(t, "abc-def", c.Get("trace_id"))
}

func TestHealth_Struct(t *testing.T) {
	// Test the Health struct directly
	tests := []struct {
		name      string
		status    string
		timestamp time.Time
	}{
		{
			name:      "creates valid health struct",
			status:    "OK",
			timestamp: time.Now().UTC(),
		},
		{
			name:      "handles different status",
			status:    "ERROR",
			timestamp: time.Now().UTC(),
		},
		{
			name:      "handles past timestamp",
			status:    "OK",
			timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := middlewares.Health{
				Status:    tt.status,
				Timestamp: tt.timestamp,
			}

			assert.Equal(t, tt.status, health.Status)
			assert.Equal(t, tt.timestamp, health.Timestamp)

			// Test JSON serialization
			jsonData, err := json.Marshal(health)
			require.NoError(t, err)

			var unmarshaled middlewares.Health
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, health.Status, unmarshaled.Status)
			// Timestamps might differ slightly due to JSON serialization, compare with tolerance
			timeDiff := health.Timestamp.Sub(unmarshaled.Timestamp)
			assert.Less(t, timeDiff, time.Second)
		})
	}
}
