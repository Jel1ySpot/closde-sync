package xray

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func parseProxyURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URI: %w", err)
	}
	return parsed, nil
}

func decodeBase64(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "-", "+")
	value = strings.ReplaceAll(value, "_", "/")
	if remainder := len(value) % 4; remainder != 0 {
		value += strings.Repeat("=", 4-remainder)
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return string(decoded), nil
	}

	decoded, err = base64.URLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func trimURIFragmentAndQuery(value string) string {
	if index := strings.IndexByte(value, '#'); index >= 0 {
		value = value[:index]
	}
	if index := strings.IndexByte(value, '?'); index >= 0 {
		value = value[:index]
	}
	return value
}

func splitHostPort(value string) (string, int, error) {
	index := strings.LastIndex(value, ":")
	if index <= 0 || index == len(value)-1 {
		return "", 0, fmt.Errorf("invalid host:port %q", value)
	}

	port, err := strconv.Atoi(value[index+1:])
	if err != nil {
		return "", 0, err
	}
	return value[:index], port, nil
}

func hostPort(parsed *url.URL) (int, error) {
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		return 0, fmt.Errorf("invalid port in proxy URI: %w", err)
	}
	return port, nil
}

func stringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func intFromAny(value any) (int, error) {
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case json.Number:
		return strconv.Atoi(typed.String())
	case string:
		return strconv.Atoi(strings.TrimSpace(typed))
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

func intOrDefault(value any, fallback int) int {
	resolved, err := intFromAny(value)
	if err != nil {
		return fallback
	}
	return resolved
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
