package xray

import (
	"encoding/json"
	"fmt"
	"net/url"

	xrayknife "github.com/lilendian0x00/xray-knife/v9/pkg/core/xray"
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
	normalized, err := normalizeProxyURI(proxyURI)
	if err != nil {
		return nil, fmt.Errorf("normalize proxy URI: %w", err)
	}

	proto, err := createXrayProtocol(normalized)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URI: %w", err)
	}
	if err := proto.Parse(); err != nil {
		return nil, fmt.Errorf("parse proxy URI: %w", err)
	}

	outboundConf, err := proto.BuildOutboundDetourConfig(false)
	if err != nil {
		return nil, fmt.Errorf("build outbound: %w", err)
	}

	data, err := json.Marshal(outboundConf)
	if err != nil {
		return nil, fmt.Errorf("marshal outbound: %w", err)
	}
	var outbound map[string]any
	if err := json.Unmarshal(data, &outbound); err != nil {
		return nil, fmt.Errorf("decode outbound: %w", err)
	}
	outbound["tag"] = outboundTag
	return outbound, nil
}

func createXrayProtocol(link string) (xrayknife.Protocol, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "vmess":
		return xrayknife.NewVmess(link), nil
	case "vless":
		return xrayknife.NewVless(link), nil
	case "ss":
		return xrayknife.NewShadowsocks(link), nil
	case "trojan":
		return xrayknife.NewTrojan(link), nil
	case "socks":
		return xrayknife.NewSocks(link), nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}

func xrayLogLevel(debug bool) string {
	if debug {
		return "debug"
	}
	return "none"
}
