package xray

import (
	"fmt"
	"net/url"
	"strings"
)

func buildURLBasedOutbound(parsed *url.URL) (map[string]any, error) {
	switch parsed.Scheme {
	case "vless":
		return buildVLESSOutbound(parsed)
	case "trojan":
		return buildTrojanOutbound(parsed)
	case "socks", "socks5":
		return buildSocksOutbound(parsed)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", parsed.Scheme)
	}
}

func buildVLESSOutbound(parsed *url.URL) (map[string]any, error) {
	port, err := hostPort(parsed)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	user := map[string]any{
		"id":         parsed.User.Username(),
		"encryption": "none",
	}
	if flow := strings.TrimSpace(query.Get("flow")); flow != "" {
		user["flow"] = flow
	}

	return map[string]any{
		"tag":      outboundTag,
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []any{
				map[string]any{
					"address": parsed.Hostname(),
					"port":    port,
					"users":   []any{user},
				},
			},
		},
		"streamSettings": newStreamSettings(
			defaultString(query.Get("type"), "tcp"),
			defaultString(query.Get("security"), "none"),
			query,
		),
	}, nil
}

func buildTrojanOutbound(parsed *url.URL) (map[string]any, error) {
	port, err := hostPort(parsed)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	password, _ := parsed.User.Password()
	if password == "" {
		password = parsed.User.Username()
	}

	return map[string]any{
		"tag":      outboundTag,
		"protocol": "trojan",
		"settings": map[string]any{
			"servers": []any{
				map[string]any{
					"address":  parsed.Hostname(),
					"port":     port,
					"password": password,
				},
			},
		},
		"streamSettings": newStreamSettings(
			defaultString(query.Get("type"), "tcp"),
			defaultString(query.Get("security"), "tls"),
			query,
		),
	}, nil
}

func buildSocksOutbound(parsed *url.URL) (map[string]any, error) {
	port, err := hostPort(parsed)
	if err != nil {
		return nil, err
	}

	server := map[string]any{
		"address": parsed.Hostname(),
		"port":    port,
	}
	if parsed.User != nil {
		server["user"] = parsed.User.Username()
		if password, ok := parsed.User.Password(); ok {
			server["pass"] = password
		}
	}

	return map[string]any{
		"tag":      outboundTag,
		"protocol": "socks",
		"settings": map[string]any{"servers": []any{server}},
	}, nil
}
