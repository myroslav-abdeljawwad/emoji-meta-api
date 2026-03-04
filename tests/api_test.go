package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"emoji-meta-api/api"
	"emoji-meta-api/models"
)

// Myroslav Mokhammad Abdeljawwad
// This test suite verifies the public API, authentication, and rate‑limiting logic of the emoji‑meta‑api.

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	router := mux.NewRouter()

	// Register all routes from the main application.
	api.RegisterRoutes(router)

	return router
}

// helper to perform a request against the test server
func doRequest(t *testing.T, handler http.Handler, method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestGetEmojiMetadata_Valid(t *testing.T) {
	server := newTestServer(t)

	// Use a valid emoji character.
	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	resp := doRequest(t, server, http.MethodGet, path, headers, nil)

	assert.Equal(t, http.StatusOK, resp.Code, "expected 200 OK")

	var meta models.Emoji
	err := json.NewDecoder(resp.Body).Decode(&meta)
	require.NoError(t, err, "response body should decode to Emoji struct")

	// Basic sanity checks on the returned metadata.
	assert.Equal(t, "😀", meta.Char, "emoji character should match")
	assert.NotEmpty(t, meta.Name, "name must be present")
}

func TestGetEmojiMetadata_Invalid(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/invalid"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	resp := doRequest(t, server, http.MethodGet, path, headers, nil)

	assert.Equal(t, http.StatusBadRequest, resp.Code, "expected 400 for invalid emoji")
}

func TestRateLimiter_BurstLimit(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	var successCount int32
	var failureCount int32

	// Assuming the limit is 5 requests per second; send 10 requests in quick succession.
	for i := 0; i < 10; i++ {
		resp := doRequest(t, server, http.MethodGet, path, headers, nil)
		if resp.Code == http.StatusOK {
			atomic.AddInt32(&successCount, 1)
		} else if resp.Code == http.StatusTooManyRequests {
			atomic.AddInt32(&failureCount, 1)
		}
	}

	assert.GreaterOrEqual(t, successCount, int32(5), "should allow at least the burst limit")
	assert.Greater(t, failureCount, int32(0), "some requests should be throttled")
}

func TestAuth_MissingToken(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{} // no Authorization header
	resp := doRequest(t, server, http.MethodGet, path, headers, nil)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "expected 401 when token is missing")
}

func TestAuth_InvalidToken(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer invalidtoken",
	}
	resp := doRequest(t, server, http.MethodGet, path, headers, nil)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "expected 401 for malformed token")
}

func TestConcurrentRequests_ConcurrencySafety(t *testing.T) {
	server := newTestServer(t)
	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	const workers = 20
	const requestsPerWorker = 5

	doneCh := make(chan struct{})
	for w := 0; w < workers; w++ {
		go func() {
			for r := 0; r < requestsPerWorker; r++ {
				resp := doRequest(t, server, http.MethodGet, path, headers, nil)
				require.True(t, resp.Code == http.StatusOK || resp.Code == http.StatusTooManyRequests,
					"expected either success or throttled")
			}
			doneCh <- struct{}{}
		}()
	}

	timeout := time.After(5 * time.Second)
	for i := 0; i < workers; i++ {
		select {
		case <-doneCh:
			continue
		case <-timeout:
			t.Fatal("test timed out, possible deadlock or unhandled error")
		}
	}
}

// TestRateLimiter_ExponentialBackoff verifies that the limiter resets after the wait period.
func TestRateLimiter_Reset(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}

	// Saturate the bucket first
	for i := 0; i < 10; i++ {
		resp := doRequest(t, server, http.MethodGet, path, headers, nil)
		if resp.Code == http.StatusTooManyRequests {
			break
		}
	}

	// Wait for limiter to reset (assuming 1 second window)
	time.Sleep(2 * time.Second)

	resp := doRequest(t, server, http.MethodGet, path, headers, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "request after reset should succeed")
}

// TestHandler_Headers verifies that content type and cache control headers are set appropriately.
func TestHandler_Headers(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	resp := doRequest(t, server, http.MethodGet, path, headers, nil)

	assert.Equal(t, "application/json; charset=utf-8", resp.Header().Get("Content-Type"))
	cacheControl := resp.Header().Get("Cache-Control")
	assert.NotEmpty(t, cacheControl, "Cache-Control header should be present")
	if strings.HasPrefix(cacheControl, "max-age=") {
		t.Logf("cache-control: %s", cacheControl)
	}
}

// TestHandler_InvalidMethod ensures that unsupported HTTP methods are rejected.
func TestHandler_InvalidMethod(t *testing.T) {
	server := newTestServer(t)

	path := "/emoji/😀"
	headers := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	resp := doRequest(t, server, http.MethodPost, path, headers, nil)

	assert.Equal(t, http.StatusMethodNotAllowed, resp.Code)
}