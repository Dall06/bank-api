package requestor

import (
	"bank-api/pkg/errs"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

type Client struct {
	http *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{http: httpClient}
}

// Response agrupa el resultado para no tener que retornar múltiples variables en la firma.
type Response struct {
	Bytes      []byte
	StatusCode int
}

// Do ejecuta la petición. Retorna solo la respuesta compacta y el error.
func (c *Client) Do(ctx context.Context, req *http.Request, withRetry bool) (*Response, error) {
	if !withRetry {
		return c.executeSingle(req)
	}
	return c.executeWithRetry(ctx, req)
}

func (c *Client) executeSingle(req *http.Request) (*Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{Bytes: body, StatusCode: resp.StatusCode}, nil
}

func (c *Client) executeWithRetry(ctx context.Context, req *http.Request) (*Response, error) {
	maxRetries := 3
	backoff := 200 * time.Millisecond

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, errs.InternalError("error reading request body for retry: %v", err)
		}
		req.Body.Close()
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = c.http.Do(req)
		if err != nil {
			if attempt == maxRetries {
				break
			}
			backoff = c.sleep(ctx, backoff)
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			if attempt == maxRetries {
				return nil, errs.ServiceUnavailableError("service down: status %d", resp.StatusCode)
			}
			backoff = c.sleep(ctx, backoff)
			continue
		}

		break
	}

	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return nil, err
		}
		return nil, errs.ServiceUnavailableError("call failed after retries: %v", err)
	}

	defer resp.Body.Close()
	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errs.InternalError("failed to read response: %v", readErr)
	}

	return &Response{Bytes: respBytes, StatusCode: resp.StatusCode}, nil
}

func (c *Client) sleep(ctx context.Context, dur time.Duration) time.Duration {
	select {
	case <-ctx.Done():
	case <-time.After(dur):
	}
	return dur * 2
}
