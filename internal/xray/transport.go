package xray

import "net/url"

func newStreamSettings(network string, security string, query url.Values) map[string]any {
	streamSettings := map[string]any{
		"network":  network,
		"security": security,
	}
	applyTransportSettings(streamSettings, network, security, query)
	return streamSettings
}

func applyTransportSettings(streamSettings map[string]any, network string, security string, query url.Values) {
	switch network {
	case "ws":
		wsSettings := map[string]any{}
		if path := firstNonEmpty(query.Get("path"), query.Get("serviceName")); path != "" {
			wsSettings["path"] = path
		}
		if host := query.Get("host"); host != "" {
			wsSettings["headers"] = map[string]any{"Host": host}
		}
		streamSettings["wsSettings"] = wsSettings
	case "grpc":
		if serviceName := firstNonEmpty(query.Get("serviceName"), query.Get("path")); serviceName != "" {
			streamSettings["grpcSettings"] = map[string]any{"serviceName": serviceName}
		}
	}

	switch security {
	case "tls":
		tlsSettings := map[string]any{}
		if sni := query.Get("sni"); sni != "" {
			tlsSettings["serverName"] = sni
		}
		if fp := query.Get("fp"); fp != "" {
			tlsSettings["fingerprint"] = fp
		}
		if alpn := splitCSV(query.Get("alpn")); len(alpn) > 0 {
			tlsSettings["alpn"] = alpn
		}
		streamSettings["tlsSettings"] = tlsSettings
	case "reality":
		realitySettings := map[string]any{}
		for key, field := range map[string]string{"sni": "serverName", "fp": "fingerprint", "pbk": "publicKey", "sid": "shortId", "spx": "spiderX"} {
			if value := query.Get(key); value != "" {
				realitySettings[field] = value
			}
		}
		streamSettings["realitySettings"] = realitySettings
	}
}
