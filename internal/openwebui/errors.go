package openwebui

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPError is returned for a non-2xx response. Status is exposed so callers
// can implement the D5 graceful-degradation path (e.g. chats endpoints
// unavailable -> fall back to completions-only mode).
type HTTPError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body != "" {
		return fmt.Sprintf("%s %s: %d: %s", e.Method, e.Path, e.Status, body)
	}
	return fmt.Sprintf("%s %s: %d", e.Method, e.Path, e.Status)
}

// statusError builds an *HTTPError from a response, draining a bounded slice of
// the body for diagnostics.
func statusError(method, path string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &HTTPError{Method: method, Path: path, Status: resp.StatusCode, Body: string(body)}
}

// ChatUnavailable reports whether err indicates the chats API is not available
// (e.g. 404/501 on a /chats endpoint), which triggers the D5 fallback to a
// completions-only client (models + stream, no session sync).
func ChatUnavailable(err error) bool {
	he, ok := err.(*HTTPError)
	if !ok {
		return false
	}
	return strings.Contains(he.Path, "/chats") &&
		(he.Status == http.StatusNotFound || he.Status == http.StatusNotImplemented)
}
