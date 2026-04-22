package xray

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func buildVMessOutbound(proxyURI string) (map[string]any, error) {
	payload, err := parseVMessPayload(proxyURI)
	if err != nil {
		return nil, err
	}

	port, err := intFromAny(payload["port"])
	if err != nil {
		return nil, fmt.Errorf("invalid vmess port: %w", err)
	}

	network := defaultString(stringField(payload, "net"), "tcp")
	security := resolveVMessSecurity(payload)
	streamSettings := newStreamSettings(network, security, queryValuesFromMap(payload))

	return map[string]any{
		"tag":      outboundTag,
		"protocol": "vmess",
		"settings": map[string]any{
			"vnext": []any{
				map[string]any{
					"address": stringField(payload, "add"),
					"port":    port,
					"users": []any{
						map[string]any{
							"id":       stringField(payload, "id"),
							"alterId":  intOrDefault(payload["aid"], 0),
							"security": defaultString(stringField(payload, "scy"), "auto"),
						},
					},
				},
			},
		},
		"streamSettings": streamSettings,
	}, nil
}

func parseVMessPayload(proxyURI string) (map[string]any, error) {
	raw := strings.TrimPrefix(proxyURI, "vmess://")
	decoded, err := decodeBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("decode vmess URI: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return nil, fmt.Errorf("parse vmess payload: %w", err)
	}
	return payload, nil
}

func queryValuesFromMap(payload map[string]any) url.Values {
	values := url.Values{}
	for _, key := range []string{"path", "host", "serviceName", "sni", "alpn", "fp", "pbk", "sid", "spx"} {
		if value := stringField(payload, key); value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func resolveVMessSecurity(payload map[string]any) string {
	if tlsValue := strings.TrimSpace(stringField(payload, "tls")); tlsValue == "tls" {
		return "tls"
	}
	if security := strings.TrimSpace(stringField(payload, "security")); security != "" {
		return security
	}
	return "none"
}
