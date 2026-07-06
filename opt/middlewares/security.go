package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// SQL injection keywords to detect
var sqlKeywords = []string{
	"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "DROP ",
	"UNION ", "OR 1=1", "AND 1=1", "' OR '", "' AND '",
	"/*", "*/", ";--", "'--",
	"EXEC ", "EXECUTE ", "TRUNCATE ", "ALTER ",
	"INFORMATION_SCHEMA", "LOAD_FILE", "INTO OUTFILE",
	"SLEEP(", "BENCHMARK(", "PG_SLEEP(",
	"WAITFOR ", "DELAY '",
}

// XSS patterns to detect
var xssKeywords = []string{
	"<script", "</script", "javascript:",
	"onerror=", "onclick=", "onload=", "onmouseover=",
	"<iframe", "<object", "<embed",
	"vbscript:", "expression(",
}

// Path traversal patterns
var pathTraversalKeywords = []string{
	"../", "..\\",
	"%2e%2e", "%252e",
	"/etc/passwd", "/proc/self",
	"\\windows\\",
}

// Headers that should NOT be scanned (browser-generated, known safe)
var skipHeaders = map[string]bool{
	"authorization":         true,
	"user-agent":            true,
	"accept":                true,
	"accept-encoding":       true,
	"accept-language":       true,
	"connection":            true,
	"host":                  true,
	"origin":                true,
	"referer":               true,
	"sec-ch-ua":             true,
	"sec-ch-ua-mobile":      true,
	"sec-ch-ua-platform":    true,
	"sec-fetch-dest":        true,
	"sec-fetch-mode":        true,
	"sec-fetch-site":        true,
	"content-type":          true,
	"content-length":        true,
	"cache-control":         true,
	"pragma":                true,
	"cookie":                true,
	"if-none-match":         true,
	"if-modified-since":     true,
	"upgrade-insecure-requests": true,
}

// Paths where body should NOT be scanned (contain passwords/sensitive data or base64 images)
var skipBodyPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/reset-password",
	"/api/v1/auth/forgot-password",
}

// Path patterns that may contain base64 images (instructor photos, settings logos)
var skipBodyContains = []string{
	"/instructors",
	"/settings/logo",
}

// shouldSkipBodyScan checks if body scanning should be skipped for this path
func shouldSkipBodyScan(path string) bool {
	for _, p := range skipBodyPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	// Check for paths that may contain base64 images
	for _, p := range skipBodyContains {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// ScanRequestMiddleware scans incoming requests for SQL injection, XSS, and path traversal
func ScanRequestMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip OPTIONS requests (CORS preflight)
			if c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Skip health checks
			if c.Request().URL.Path == "/health" {
				return next(c)
			}

			// Scan URL path only (not full URL to avoid encoded data issues)
			urlPath := c.Request().URL.Path
			if blocked := scanForThreats(urlPath); blocked != "" {
				return blockRequest(c, blocked, "url")
			}

			// Scan query params
			for key, values := range c.QueryParams() {
				if blocked := scanForThreats(key); blocked != "" {
					return blockRequest(c, blocked, "query_key")
				}
				for _, val := range values {
					if blocked := scanForThreats(val); blocked != "" {
						return blockRequest(c, blocked, "query_value")
					}
				}
			}

			// Scan custom headers only (skip browser-generated headers)
			for key, values := range c.Request().Header {
				if skipHeaders[strings.ToLower(key)] {
					continue
				}
				for _, val := range values {
					if blocked := scanForThreats(val); blocked != "" {
						return blockRequest(c, blocked, "header")
					}
				}
			}

			// Scan body for POST/PUT/PATCH (skip auth endpoints to avoid password false positives)
			if !shouldSkipBodyScan(c.Request().URL.Path) {
				if c.Request().Method == http.MethodPost ||
					c.Request().Method == http.MethodPut ||
					c.Request().Method == http.MethodPatch {

					if c.Request().Body != nil && c.Request().ContentLength > 0 {
						body, err := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20)) // 1MB max
						if err != nil {
							return echo.NewHTTPError(http.StatusBadRequest, "failed to read body")
						}
						c.Request().Body = io.NopCloser(bytes.NewReader(body))

						if blocked := scanForThreats(string(body)); blocked != "" {
							return blockRequest(c, blocked, "body")
						}
					}
				}
			}

			return next(c)
		}
	}
}

// scanForThreats checks input for malicious patterns
func scanForThreats(input string) string {
	if input == "" {
		return ""
	}

	// Normalize input (uppercase + decode common URL encoding)
	normalized := strings.ToUpper(input)
	normalized = strings.ReplaceAll(normalized, "%27", "'")
	normalized = strings.ReplaceAll(normalized, "%22", "\"")
	normalized = strings.ReplaceAll(normalized, "%3C", "<")
	normalized = strings.ReplaceAll(normalized, "%3E", ">")
	normalized = strings.ReplaceAll(normalized, "%2F", "/")
	normalized = strings.ReplaceAll(normalized, "%5C", "\\")

	// Check SQL injection
	for _, keyword := range sqlKeywords {
		if strings.Contains(normalized, strings.ToUpper(keyword)) {
			return "sql_injection"
		}
	}

	// Check XSS
	for _, keyword := range xssKeywords {
		if strings.Contains(normalized, strings.ToUpper(keyword)) {
			return "xss"
		}
	}

	// Check path traversal
	for _, keyword := range pathTraversalKeywords {
		if strings.Contains(normalized, strings.ToUpper(keyword)) {
			return "path_traversal"
		}
	}

	return ""
}

// blockRequest returns 403 Forbidden
func blockRequest(c echo.Context, threat, location string) error {
	c.Logger().Warnf("Blocked %s in %s from %s: %s", threat, location, c.RealIP(), c.Request().URL.Path)
	return echo.NewHTTPError(http.StatusForbidden, map[string]string{
		"error":   "blocked",
		"message": "Request blocked by security policy",
	})
}
