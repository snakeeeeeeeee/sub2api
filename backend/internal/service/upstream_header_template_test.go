package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyAccountUpstreamHeadersRendersStaticAndDynamicValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newUpstreamHeaderTemplateTestContext(t, "/v1/messages?trace_id=query-7", []byte(`{
		"metadata": {"tenant_id": "tenant-a"},
		"tags": ["a", "b"],
		"enabled": true
	}`))
	c.Request.Header.Set("Session-ID", "session-123")
	c.Request.Header.Set("X-NewAPI-Meta", `{"user":{"id":"user-9"}}`)

	req := httptest.NewRequest(http.MethodPost, "https://upstream.example.test/v1/messages", nil)
	account := &Account{
		ID:       42,
		Name:     "acct-a",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"routing": map[string]any{"pool": map[string]any{"name": "pool-a"}},
			accountUpstreamHeadersExtraKey: map[string]any{
				"X-Pool-Session-ID": "{{header.session-id}}",
				"X-Tenant-ID":       "{{body.metadata.tenant_id}}",
				"X-User-ID":         "{{json_header.x-newapi-meta:user.id}}",
				"X-Trace-ID":        "{{query.trace_id}}",
				"X-Pool-Source":     "sub2api",
				"X-Account-ID":      "{{account.id}}",
				"X-Account-Name":    "{{account.name}}",
				"X-Platform":        "{{account.platform}}",
				"X-Type":            "{{account.type}}",
				"X-Extra-Pool":      "{{account.extra.routing.pool.name}}",
				"X-JSON-Tags":       "{{body.tags}}",
				"X-Mixed":           "tenant={{body.metadata.tenant_id}};session={{header.session-id}}",
				"X-Boolean":         "{{body.enabled}}",
			},
		},
	}

	ApplyAccountUpstreamHeaders(req, c, account, readGinRequestBodyForTest(t, c))

	require.Equal(t, "session-123", req.Header.Get("X-Pool-Session-ID"))
	require.Equal(t, "tenant-a", req.Header.Get("X-Tenant-ID"))
	require.Equal(t, "user-9", req.Header.Get("X-User-ID"))
	require.Equal(t, "query-7", req.Header.Get("X-Trace-ID"))
	require.Equal(t, "sub2api", req.Header.Get("X-Pool-Source"))
	require.Equal(t, "42", req.Header.Get("X-Account-ID"))
	require.Equal(t, "acct-a", req.Header.Get("X-Account-Name"))
	require.Equal(t, PlatformOpenAI, req.Header.Get("X-Platform"))
	require.Equal(t, AccountTypeAPIKey, req.Header.Get("X-Type"))
	require.Equal(t, "pool-a", req.Header.Get("X-Extra-Pool"))
	require.Equal(t, `["a","b"]`, req.Header.Get("X-JSON-Tags"))
	require.Equal(t, "tenant=tenant-a;session=session-123", req.Header.Get("X-Mixed"))
	require.Equal(t, "true", req.Header.Get("X-Boolean"))
}

func TestApplyAccountUpstreamHeadersSkipsMissingValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newUpstreamHeaderTemplateTestContext(t, "/v1/messages", []byte(`{"metadata":{}}`))
	req := httptest.NewRequest(http.MethodPost, "https://upstream.example.test/v1/messages", nil)
	account := &Account{
		Extra: map[string]any{
			accountUpstreamHeadersExtraKey: map[string]string{
				"X-Missing-Header": "{{header.session-id}}",
				"X-Missing-Body":   "{{body.metadata.tenant_id}}",
				"X-Missing-Mixed":  "tenant={{body.metadata.tenant_id}}",
			},
		},
	}

	ApplyAccountUpstreamHeaders(req, c, account, readGinRequestBodyForTest(t, c))

	require.Empty(t, req.Header.Get("X-Missing-Header"))
	require.Empty(t, req.Header.Get("X-Missing-Body"))
	require.Empty(t, req.Header.Get("X-Missing-Mixed"))
}

func TestApplyAccountUpstreamHeadersRejectsForbiddenAndInvalidHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newUpstreamHeaderTemplateTestContext(t, "/v1/messages", []byte(`{"metadata":{"tenant_id":"tenant-a"}}`))
	req := httptest.NewRequest(http.MethodPost, "https://upstream.example.test/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer original")
	req.Header.Set("Content-Type", "application/json")
	account := &Account{
		Extra: map[string]any{
			accountUpstreamHeadersExtraKey: map[string]any{
				"Authorization":  "Bearer custom",
				"x-api-key":      "custom-key",
				"Content-Type":   "text/plain",
				"Bad Header":     "bad",
				"X-Safe-Header":  "safe",
				"X-Unsafe-Value": "bad\r\nInjected: true",
			},
		},
	}

	ApplyAccountUpstreamHeaders(req, c, account, readGinRequestBodyForTest(t, c))

	require.Equal(t, "Bearer original", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Empty(t, req.Header.Get("x-api-key"))
	require.Empty(t, req.Header.Get("Bad Header"))
	require.Empty(t, req.Header.Get("X-Unsafe-Value"))
	require.Equal(t, "safe", req.Header.Get("X-Safe-Header"))
}

func TestApplyAccountUpstreamHeadersSkipsRenderedCRLF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newUpstreamHeaderTemplateTestContext(t, "/v1/messages", []byte(`{"metadata":{"tenant_id":"tenant-a"}}`))
	c.Request.Header.Set("Session-ID", "abc\r\nX-Evil: yes")
	req := httptest.NewRequest(http.MethodPost, "https://upstream.example.test/v1/messages", nil)
	account := &Account{
		Extra: map[string]any{
			accountUpstreamHeadersExtraKey: map[string]any{
				"X-Pool-Session-ID": "{{header.session-id}}",
			},
		},
	}

	ApplyAccountUpstreamHeaders(req, c, account, readGinRequestBodyForTest(t, c))

	require.Empty(t, req.Header.Get("X-Pool-Session-ID"))
}

func newUpstreamHeaderTemplateTestContext(t *testing.T, target string, body []byte) *gin.Context {
	t.Helper()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	c.Set("body", body)
	return c
}

func readGinRequestBodyForTest(t *testing.T, c *gin.Context) []byte {
	t.Helper()

	raw, ok := c.Get("body")
	require.True(t, ok)
	body, ok := raw.([]byte)
	require.True(t, ok)
	return body
}
