package xray

import (
	"fmt"
	"strings"
)

type shadowsocksCredentials struct {
	Method   string
	Password string
	Host     string
	Port     int
}

func buildShadowsocksOutbound(proxyURI string) (map[string]any, error) {
	credentials, err := parseShadowsocksCredentials(proxyURI)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"tag":      outboundTag,
		"protocol": "shadowsocks",
		"settings": map[string]any{
			"servers": []any{
				map[string]any{
					"address":  credentials.Host,
					"port":     credentials.Port,
					"method":   credentials.Method,
					"password": credentials.Password,
				},
			},
		},
	}, nil
}

func parseShadowsocksCredentials(proxyURI string) (shadowsocksCredentials, error) {
	body := trimURIFragmentAndQuery(strings.TrimPrefix(proxyURI, "ss://"))
	if left, right, ok := strings.Cut(body, "@"); ok {
		userinfo := left
		if decoded, err := decodeBase64(left); err == nil {
			userinfo = decoded
		}

		credentials, err := parseShadowsocksUserInfo(userinfo)
		if err != nil {
			return shadowsocksCredentials{}, err
		}
		host, port, err := splitHostPort(right)
		if err != nil {
			return shadowsocksCredentials{}, err
		}
		credentials.Host = host
		credentials.Port = port
		return credentials, nil
	}

	decoded, err := decodeBase64(body)
	if err != nil {
		return shadowsocksCredentials{}, err
	}
	userinfo, target, ok := strings.Cut(decoded, "@")
	if !ok {
		return shadowsocksCredentials{}, fmt.Errorf("invalid shadowsocks URI")
	}

	credentials, err := parseShadowsocksUserInfo(userinfo)
	if err != nil {
		return shadowsocksCredentials{}, err
	}
	host, port, err := splitHostPort(target)
	if err != nil {
		return shadowsocksCredentials{}, err
	}
	credentials.Host = host
	credentials.Port = port
	return credentials, nil
}

func parseShadowsocksUserInfo(value string) (shadowsocksCredentials, error) {
	method, password, ok := strings.Cut(value, ":")
	if !ok {
		return shadowsocksCredentials{}, fmt.Errorf("invalid proxy user info")
	}
	return shadowsocksCredentials{Method: method, Password: password}, nil
}
