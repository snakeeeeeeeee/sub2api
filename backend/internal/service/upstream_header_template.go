package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"golang.org/x/net/http/httpguts"
)

const accountUpstreamHeadersExtraKey = "upstream_headers"

var (
	upstreamHeaderPlaceholderPattern = regexp.MustCompile(`{{\s*([^{}]+?)\s*}}`)
	forbiddenUpstreamHeaderNames     = map[string]struct{}{
		"authorization":     {},
		"cookie":            {},
		"host":              {},
		"content-length":    {},
		"connection":        {},
		"transfer-encoding": {},
		"x-api-key":         {},
		"x-goog-api-key":    {},
		"content-type":      {},
	}
)

// ApplyAccountUpstreamHeaders renders account.extra.upstream_headers and appends
// the resulting safe headers to an already-built upstream request.
func ApplyAccountUpstreamHeaders(req *http.Request, c *gin.Context, account *Account, body []byte) {
	if req == nil || account == nil {
		return
	}
	templates := accountUpstreamHeaderTemplates(account)
	if len(templates) == 0 {
		return
	}

	for name, tmpl := range templates {
		headerName := strings.TrimSpace(name)
		if !isAllowedAccountUpstreamHeaderName(headerName) {
			continue
		}
		value, ok := renderAccountUpstreamHeaderTemplate(tmpl, c, account, body)
		if !ok || strings.ContainsAny(value, "\r\n") {
			continue
		}
		req.Header.Set(headerName, value)
	}
}

func accountUpstreamHeaderTemplates(account *Account) map[string]string {
	if account == nil || account.Extra == nil {
		return nil
	}
	raw, ok := account.Extra[accountUpstreamHeadersExtraKey]
	if !ok || raw == nil {
		return nil
	}

	out := map[string]string{}
	switch v := raw.(type) {
	case map[string]string:
		for k, val := range v {
			if strings.TrimSpace(k) != "" {
				out[k] = val
			}
		}
	case map[string]any:
		for k, val := range v {
			s, ok := val.(string)
			if ok && strings.TrimSpace(k) != "" {
				out[k] = s
			}
		}
	}
	return out
}

func isAllowedAccountUpstreamHeaderName(name string) bool {
	if name == "" || !httpguts.ValidHeaderFieldName(name) {
		return false
	}
	_, forbidden := forbiddenUpstreamHeaderNames[strings.ToLower(name)]
	return !forbidden
}

func renderAccountUpstreamHeaderTemplate(tmpl string, c *gin.Context, account *Account, body []byte) (string, bool) {
	if strings.ContainsAny(tmpl, "\r\n") {
		return "", false
	}

	matches := upstreamHeaderPlaceholderPattern.FindAllStringSubmatchIndex(tmpl, -1)
	if len(matches) == 0 {
		return tmpl, true
	}

	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(tmpl) {
		expr := tmpl[matches[0][2]:matches[0][3]]
		value, ok := resolveAccountUpstreamHeaderPlaceholder(expr, c, account, body)
		if !ok {
			return "", false
		}
		return accountUpstreamHeaderValueToString(value), true
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		_, _ = b.WriteString(tmpl[last:match[0]])
		expr := tmpl[match[2]:match[3]]
		value, ok := resolveAccountUpstreamHeaderPlaceholder(expr, c, account, body)
		if !ok {
			return "", false
		}
		_, _ = b.WriteString(accountUpstreamHeaderValueToString(value))
		last = match[1]
	}
	_, _ = b.WriteString(tmpl[last:])
	return b.String(), true
}

func resolveAccountUpstreamHeaderPlaceholder(expr string, c *gin.Context, account *Account, body []byte) (any, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, false
	}

	switch {
	case strings.HasPrefix(expr, "header."):
		name := strings.TrimSpace(strings.TrimPrefix(expr, "header."))
		if name == "" || c == nil {
			return nil, false
		}
		v := strings.TrimSpace(c.GetHeader(name))
		if v == "" {
			return nil, false
		}
		return v, true
	case strings.HasPrefix(expr, "query."):
		name := strings.TrimSpace(strings.TrimPrefix(expr, "query."))
		if name == "" || c == nil || c.Request == nil {
			return nil, false
		}
		v := strings.TrimSpace(c.Query(name))
		if v == "" {
			return nil, false
		}
		return v, true
	case strings.HasPrefix(expr, "body."):
		path := strings.TrimSpace(strings.TrimPrefix(expr, "body."))
		return resolveJSONPathValue(body, path)
	case strings.HasPrefix(expr, "json_header."):
		spec := strings.TrimSpace(strings.TrimPrefix(expr, "json_header."))
		headerName, path, ok := strings.Cut(spec, ":")
		if !ok || strings.TrimSpace(headerName) == "" || strings.TrimSpace(path) == "" || c == nil {
			return nil, false
		}
		raw := strings.TrimSpace(c.GetHeader(strings.TrimSpace(headerName)))
		if raw == "" {
			return nil, false
		}
		return resolveJSONPathValue([]byte(raw), strings.TrimSpace(path))
	case strings.HasPrefix(expr, "account.extra."):
		path := strings.TrimSpace(strings.TrimPrefix(expr, "account.extra."))
		if account == nil || account.Extra == nil || path == "" {
			return nil, false
		}
		raw, err := json.Marshal(account.Extra)
		if err != nil {
			return nil, false
		}
		return resolveJSONPathValue(raw, path)
	case strings.HasPrefix(expr, "account."):
		field := strings.TrimSpace(strings.TrimPrefix(expr, "account."))
		if account == nil {
			return nil, false
		}
		switch field {
		case "id":
			return account.ID, true
		case "name":
			if account.Name == "" {
				return nil, false
			}
			return account.Name, true
		case "platform":
			if account.Platform == "" {
				return nil, false
			}
			return account.Platform, true
		case "type":
			if account.Type == "" {
				return nil, false
			}
			return account.Type, true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}

func resolveJSONPathValue(raw []byte, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if len(raw) == 0 || path == "" || !gjson.ValidBytes(raw) {
		return nil, false
	}
	result := gjson.GetBytes(raw, path)
	if !result.Exists() || result.Type == gjson.Null {
		return nil, false
	}
	return result, true
}

func accountUpstreamHeaderValueToString(value any) string {
	switch v := value.(type) {
	case gjson.Result:
		switch v.Type {
		case gjson.JSON:
			return compactJSONRaw(v.Raw)
		case gjson.String:
			return v.String()
		case gjson.Number:
			return v.Raw
		case gjson.True:
			return "true"
		case gjson.False:
			return "false"
		case gjson.Null:
			return ""
		default:
			return v.String()
		}
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func compactJSONRaw(raw string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(raw)); err == nil {
		return buf.String()
	}
	return raw
}
