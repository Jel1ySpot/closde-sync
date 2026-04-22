package xray

import (
	"encoding/json"
	"strings"
)

const outboundTag = "proxy"

func BuildConfigJSON(proxyURI string, listenHost string, listenPort int, debug bool) (string, error) {
	config, err := buildConfigDocument(proxyURI, listenHost, listenPort, debug)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func buildConfigDocument(proxyURI string, listenHost string, listenPort int, debug bool) (map[string]any, error) {
	outbound, err := buildOutbound(proxyURI)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"log": map[string]any{"loglevel": xrayLogLevel(debug)},
		"inbounds": []any{
			map[string]any{
				"listen":   listenHost,
				"port":     listenPort,
				"protocol": "http",
				"settings": map[string]any{},
			},
		},
		"outbounds": []any{outbound},
	}, nil
}

func buildOutbound(proxyURI string) (map[string]any, error) {
	switch {
	case strings.HasPrefix(proxyURI, "vmess://"):
		return buildVMessOutbound(proxyURI)
	case strings.HasPrefix(proxyURI, "ss://"):
		return buildShadowsocksOutbound(proxyURI)
	default:
		parsed, err := parseProxyURL(proxyURI)
		if err != nil {
			return nil, err
		}
		return buildURLBasedOutbound(parsed)
	}
}

func xrayLogLevel(debug bool) string {
	if debug {
		return "debug"
	}
	return "none"
}
