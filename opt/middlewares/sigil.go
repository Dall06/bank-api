package middlewares

import (
	"io"
	"net/http"
	"strings"

	"bank-api/pkg/sigil"

	"github.com/labstack/echo/v4"
)

// Public paths that should skip sigil verification (no service-to-service auth required)
var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/validate-code",
	"/api/v1/auth/forgot-password",
	"/api/v1/auth/reset-password",
	"/api/v1/plans",
	"/api/v1/companies/search",
	"/api/v1/payments/create-intent",
	"/api/v1/webhooks/stripe",
}

// NewSigilVerifier returns an Echo middleware that verifies sigil signatures
// on incoming requests from other services (e.g., gateway -> companies/gym)
func NewSigilVerifier(verifier *sigil.Verifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip health checks - must be public for Docker health checks
			if strings.HasSuffix(path, "/health") {
				return next(c)
			}

			// Skip public paths that don't require service-to-service auth
			for _, p := range publicPaths {
				if path == p {
					return next(c)
				}
			}

			// Get sigil headers
			serviceID := c.Request().Header.Get(sigil.HeaderServiceID)
			timestamp := c.Request().Header.Get(sigil.HeaderTimestamp)
			signature := c.Request().Header.Get(sigil.HeaderSignature)

			// Read body for verification
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
			}

			// Restore body for downstream handlers
			c.Request().Body = io.NopCloser(io.MultiReader(
				// Use bytes.NewReader to create a new reader from the body
				&bytesReader{data: body},
			))

			// Verify the request
			if err := verifier.VerifyRequest(serviceID, timestamp, signature, body); err != nil {
				switch err {
				case sigil.ErrMissingHeaders:
					return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication headers")
				case sigil.ErrUnknownService:
					return echo.NewHTTPError(http.StatusForbidden, "unknown service")
				case sigil.ErrTimestampExpired:
					return echo.NewHTTPError(http.StatusUnauthorized, "request expired")
				case sigil.ErrInvalidSignature:
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
				default:
					return echo.NewHTTPError(http.StatusUnauthorized, "authentication failed")
				}
			}

			// Set service ID in context for logging/auditing
			c.Set("service_id", serviceID)

			return next(c)
		}
	}
}

// bytesReader wraps a byte slice to implement io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// SigilHeaders is a helper to add sigil headers to outgoing requests
type SigilHeaders struct {
	signer *sigil.Signer
}

// NewSigilHeaders creates a helper for adding sigil headers
func NewSigilHeaders(signer *sigil.Signer) *SigilHeaders {
	return &SigilHeaders{signer: signer}
}

// AddHeaders adds sigil headers to an HTTP request
func (h *SigilHeaders) AddHeaders(req *http.Request, body []byte) {
	headers := h.signer.SignRequest(body)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}
