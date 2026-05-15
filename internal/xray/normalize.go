package xray

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func normalizeProxyURI(raw string) (string, error) {
	switch {
	case strings.HasPrefix(raw, "ss://"):
		return normalizeSSURI(raw)
	case strings.HasPrefix(raw, "vless://"):
		return normalizeVLESSURI(raw)
	case strings.HasPrefix(raw, "vmess://"):
		return normalizeVMessURI(raw)
	default:
		return raw, nil
	}
}

func normalizeSSURI(raw string) (string, error) {
	body, fragment := splitFragment(strings.TrimPrefix(raw, "ss://"))

	var userinfo, target string
	if idx := strings.LastIndex(body, "@"); idx >= 0 {
		userinfo = body[:idx]
		target = body[idx+1:]
		if decoded, err := decodeBase64(userinfo); err == nil && strings.Contains(decoded, ":") {
			userinfo = decoded
		}
	} else {
		decoded, err := decodeBase64(body)
		if err != nil {
			return "", fmt.Errorf("decode ss body: %w", err)
		}
		idx := strings.LastIndex(decoded, "@")
		if idx < 0 {
			return "", fmt.Errorf("invalid ss body")
		}
		userinfo = decoded[:idx]
		target = decoded[idx+1:]
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(userinfo))
	return fmt.Sprintf("ss://%s@%s%s", encoded, target, fragment), nil
}

func normalizeVLESSURI(raw string) (string, error) {
	body, fragment := splitFragment(strings.TrimPrefix(raw, "vless://"))

	pivot := body
	if i := strings.IndexByte(pivot, '?'); i >= 0 {
		pivot = pivot[:i]
	}
	if strings.ContainsRune(pivot, '@') {
		return raw, nil
	}

	var query string
	if i := strings.IndexByte(body, '?'); i >= 0 {
		query = body[i+1:]
		body = body[:i]
	}
	decoded, err := decodeBase64(body)
	if err != nil {
		return "", fmt.Errorf("decode vless body: %w", err)
	}
	userinfo, target, ok := strings.Cut(decoded, "@")
	if !ok {
		return "", fmt.Errorf("invalid vless body")
	}
	_, id, ok := strings.Cut(userinfo, ":")
	if !ok {
		id = userinfo
	}

	values, err := url.ParseQuery(query)
	if err != nil {
		return "", fmt.Errorf("parse vless query: %w", err)
	}
	rewriteAliceQuery(values)

	rendered := values.Encode()
	if rendered != "" {
		rendered = "?" + rendered
	}
	return fmt.Sprintf("vless://%s@%s%s%s", id, target, rendered, fragment), nil
}

func normalizeVMessURI(raw string) (string, error) {
	body, fragment := splitFragment(strings.TrimPrefix(raw, "vmess://"))

	qIdx := strings.IndexByte(body, '?')
	if qIdx < 0 {
		return raw, nil
	}

	payload := body[:qIdx]
	query := body[qIdx+1:]

	decoded, err := decodeBase64(payload)
	if err != nil {
		return "", fmt.Errorf("decode vmess body: %w", err)
	}
	userinfo, target, ok := strings.Cut(decoded, "@")
	if !ok {
		return "", fmt.Errorf("invalid vmess body")
	}
	cipher, id, ok := strings.Cut(userinfo, ":")
	if !ok {
		id = userinfo
		cipher = "auto"
	}
	host, port, err := splitHostPort(target)
	if err != nil {
		return "", err
	}

	values, err := url.ParseQuery(query)
	if err != nil {
		return "", fmt.Errorf("parse vmess query: %w", err)
	}

	alterID := 0
	if v := strings.TrimSpace(values.Get("alterId")); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			alterID = i
		}
	}
	tls := ""
	if nonZero(values.Get("tls")) {
		tls = "tls"
	}

	payloadJSON := map[string]any{
		"v":    "2",
		"ps":   values.Get("remarks"),
		"add":  host,
		"port": port,
		"id":   id,
		"aid":  alterID,
		"scy":  cipher,
		"net":  firstNonEmpty(values.Get("net"), values.Get("type"), "tcp"),
		"tls":  tls,
	}
	if sni := firstNonEmpty(values.Get("sni"), values.Get("peer")); sni != "" {
		payloadJSON["sni"] = sni
	}
	if v := values.Get("host"); v != "" {
		payloadJSON["host"] = v
	}
	if v := values.Get("path"); v != "" {
		payloadJSON["path"] = v
	}

	encoded, err := json.Marshal(payloadJSON)
	if err != nil {
		return "", fmt.Errorf("marshal vmess json: %w", err)
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(encoded) + fragment, nil
}

func rewriteAliceQuery(values url.Values) {
	if values.Get("type") == "" {
		values.Set("type", "tcp")
	}
	if values.Get("security") == "" {
		switch {
		case nonZero(values.Get("xtls")) && values.Get("pbk") != "":
			values.Set("security", "reality")
		case nonZero(values.Get("tls")):
			values.Set("security", "tls")
		default:
			values.Set("security", "none")
		}
	}
	if values.Get("sni") == "" {
		if peer := values.Get("peer"); peer != "" {
			values.Set("sni", peer)
		}
	}
	values.Del("peer")
	values.Del("tls")
	values.Del("xtls")
	values.Del("remarks")
}

func splitFragment(value string) (string, string) {
	if i := strings.IndexByte(value, '#'); i >= 0 {
		return value[:i], value[i:]
	}
	return value, ""
}

func splitHostPort(value string) (string, int, error) {
	idx := strings.LastIndex(value, ":")
	if idx <= 0 || idx == len(value)-1 {
		return "", 0, fmt.Errorf("invalid host:port %q", value)
	}
	port, err := strconv.Atoi(value[idx+1:])
	if err != nil {
		return "", 0, err
	}
	return value[:idx], port, nil
}

func decodeBase64(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "-", "+")
	value = strings.ReplaceAll(value, "_", "/")
	if remainder := len(value) % 4; remainder != 0 {
		value += strings.Repeat("=", 4-remainder)
	}

	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded), nil
	}
	decoded, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func nonZero(value string) bool {
	v := strings.TrimSpace(value)
	return v != "" && v != "0"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
