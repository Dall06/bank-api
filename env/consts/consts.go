package consts

type contextKey string

const (
	IdempotencyKeyContextKey contextKey = "idempotency_key"
	BypassCacheContextKey    contextKey = "bypass_cache"
	MockIDContextKey         contextKey = "mock_id"
)

const (
	MockIDHeaderKey = "X-Mock-Id"
)
