package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpstreamHeaderTemplatesReachAnthropicAPIKeyPassthroughRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-test","metadata":{"tenant_id":"tenant-a"},"messages":[]}`)
	c := newUpstreamHeaderTemplateBuilderContext(t, "/v1/messages", body)
	c.Request.Header.Set("Session-ID", "session-123")

	account := accountWithTemplateHeaders(11, PlatformAnthropic, AccountTypeAPIKey)
	account.Credentials = map[string]any{"base_url": "https://anthropic.example.test"}

	req, wireBody, err := (&GatewayService{cfg: upstreamHeaderTemplateTestConfig()}).
		buildUpstreamRequestAnthropicAPIKeyPassthrough(context.Background(), c, account, body, "anthropic-token")

	require.NoError(t, err)
	require.Equal(t, body, wireBody)
	require.Equal(t, "session-123", req.Header.Get("X-Pool-Session-ID"))
	require.Equal(t, "tenant-a", req.Header.Get("X-Tenant-ID"))
	require.Equal(t, "11", req.Header.Get("X-Account-ID"))
	require.Equal(t, "anthropic-token", getHeaderRaw(req.Header, "x-api-key"))
	require.Empty(t, getHeaderRaw(req.Header, "Authorization"))
}

func TestUpstreamHeaderTemplatesReachOpenAIAPIKeyPassthroughRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-test","metadata":{"tenant_id":"tenant-openai"}}`)
	c := newUpstreamHeaderTemplateBuilderContext(t, "/v1/responses", body)
	c.Request.Header.Set("Session-ID", "openai-session")

	account := accountWithTemplateHeaders(12, PlatformOpenAI, AccountTypeAPIKey)
	account.Credentials = map[string]any{"base_url": "https://openai.example.test"}

	req, err := (&OpenAIGatewayService{cfg: upstreamHeaderTemplateTestConfig()}).
		buildUpstreamRequestOpenAIPassthrough(context.Background(), c, account, body, "openai-token")

	require.NoError(t, err)
	require.Equal(t, "openai-session", req.Header.Get("X-Pool-Session-ID"))
	require.Equal(t, "tenant-openai", req.Header.Get("X-Tenant-ID"))
	require.Equal(t, "12", req.Header.Get("X-Account-ID"))
	require.Equal(t, "Bearer openai-token", req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("x-api-key"))
}

func TestUpstreamHeaderTemplatesReachGeminiAPIKeyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"metadata":{"tenant_id":"tenant-gemini"}}`)
	c := newUpstreamHeaderTemplateBuilderContext(t, "/v1/chat/completions", body)
	c.Request.Header.Set("Session-ID", "gemini-session")

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`)),
		},
	}
	svc := &GeminiMessagesCompatService{
		httpUpstream: httpStub,
		cfg:          upstreamHeaderTemplateTestConfig(),
	}
	account := accountWithTemplateHeaders(13, PlatformGemini, AccountTypeAPIKey)
	account.Credentials = map[string]any{
		"api_key":  "gemini-token",
		"base_url": "https://gemini.example.test",
	}
	account.Concurrency = 1

	_, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, httpStub.lastReq)
	require.Equal(t, "gemini-session", httpStub.lastReq.Header.Get("X-Pool-Session-ID"))
	require.Equal(t, "tenant-gemini", httpStub.lastReq.Header.Get("X-Tenant-ID"))
	require.Equal(t, "13", httpStub.lastReq.Header.Get("X-Account-ID"))
	require.Equal(t, "gemini-token", httpStub.lastReq.Header.Get("x-goog-api-key"))
	require.Empty(t, httpStub.lastReq.Header.Get("Authorization"))
}

func TestUpstreamHeaderTemplatesReachAntigravityUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-test","metadata":{"tenant_id":"tenant-ag"},"messages":[]}`)
	c := newUpstreamHeaderTemplateBuilderContext(t, "/v1/messages", body)
	c.Request.Header.Set("Session-ID", "ag-session")

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-test","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}
	account := accountWithTemplateHeaders(14, PlatformAntigravity, AccountTypeAPIKey)
	account.Credentials = map[string]any{
		"api_key":  "antigravity-token",
		"base_url": "https://antigravity.example.test",
	}
	account.Concurrency = 1

	_, err := (&AntigravityGatewayService{httpUpstream: httpStub}).
		ForwardUpstream(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, httpStub.lastReq)
	require.Equal(t, "ag-session", httpStub.lastReq.Header.Get("X-Pool-Session-ID"))
	require.Equal(t, "tenant-ag", httpStub.lastReq.Header.Get("X-Tenant-ID"))
	require.Equal(t, "14", httpStub.lastReq.Header.Get("X-Account-ID"))
	require.Equal(t, "Bearer antigravity-token", httpStub.lastReq.Header.Get("Authorization"))
	require.Equal(t, "antigravity-token", httpStub.lastReq.Header.Get("x-api-key"))
}

func accountWithTemplateHeaders(id int64, platform, accountType string) *Account {
	return &Account{
		ID:       id,
		Name:     "template-account",
		Platform: platform,
		Type:     accountType,
		Extra: map[string]any{
			accountUpstreamHeadersExtraKey: map[string]any{
				"X-Pool-Session-ID": "{{header.session-id}}",
				"X-Tenant-ID":       "{{body.metadata.tenant_id}}",
				"X-Account-ID":      "{{account.id}}",
				"Authorization":     "Bearer should-not-override",
				"x-api-key":         "should-not-override",
				"x-goog-api-key":    "should-not-override",
			},
		},
	}
}

func newUpstreamHeaderTemplateBuilderContext(t *testing.T, target string, body []byte) *gin.Context {
	t.Helper()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	return c
}

func upstreamHeaderTemplateTestConfig() *config.Config {
	return &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{Enabled: false},
		},
	}
}
