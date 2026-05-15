package xray

import (
	"encoding/json"
	"testing"
)

func TestBuildOutboundRecognizesURIs(t *testing.T) {
	cases := []struct {
		name     string
		uri      string
		protocol string
		address  string
		port     int
		check    func(t *testing.T, settings, stream map[string]any)
	}{
		{
			name:     "trojan with base64-looking password",
			uri:      "trojan://5pSv6YKj5Lq65rCR5YWx5ZKM5Zu95LiH5bKB@vps.jellyspot.cc:443#US,%20California",
			protocol: "trojan",
			address:  "vps.jellyspot.cc",
			port:     443,
			check: func(t *testing.T, settings, stream map[string]any) {
				server := firstServer(t, settings)
				if got := server["password"]; got != "5pSv6YKj5Lq65rCR5YWx5ZKM5Zu95LiH5bKB" {
					t.Fatalf("trojan password = %v", got)
				}
				if got := stream["security"]; got != "tls" {
					t.Fatalf("trojan security = %v", got)
				}
			},
		},
		{
			name:     "ss base64-encoded SIP002 body",
			uri:      "ss://MjAyMi1ibGFrZTMtYWVzLTI1Ni1nY206Q2NIRHMvSjZxZmV1TjlXODZ6SVFHSm1kSHArbXBWQVJWblp2YVh5ZTVtOD06VThDSEthNzZzOEF6SnBpVkpUajhQNS8wOFVnVTBGWHZ1dzlxWEoyditnbz1AOTEuMTAzLjEyMi4xNDM6MTQzMzM#alice",
			protocol: "shadowsocks",
			address:  "91.103.122.143",
			port:     14333,
			check: func(t *testing.T, settings, stream map[string]any) {
				server := firstServer(t, settings)
				if got := server["method"]; got != "2022-blake3-aes-256-gcm" {
					t.Fatalf("ss method = %v", got)
				}
				want := "CcHDs/J6qfeuN9W86zIQGJmdHp+mpVARVnZvaXye5m8=:U8CHKa76s8AzJpiVJTj8P5/08UgU0FXvuw9qXJ2v+go="
				if got := server["password"]; got != want {
					t.Fatalf("ss password = %v", got)
				}
			},
		},
		{
			name:     "ss SIP002 plain with SS2022 PSK",
			uri:      "ss://2022-blake3-aes-256-gcm:CcHDs/J6qfeuN9W86zIQGJmdHp+mpVARVnZvaXye5m8=:U8CHKa76s8AzJpiVJTj8P5/08UgU0FXvuw9qXJ2v+go=@91.103.122.143:14333#alice",
			protocol: "shadowsocks",
			address:  "91.103.122.143",
			port:     14333,
			check: func(t *testing.T, settings, stream map[string]any) {
				server := firstServer(t, settings)
				if got := server["method"]; got != "2022-blake3-aes-256-gcm" {
					t.Fatalf("ss method = %v", got)
				}
				want := "CcHDs/J6qfeuN9W86zIQGJmdHp+mpVARVnZvaXye5m8=:U8CHKa76s8AzJpiVJTj8P5/08UgU0FXvuw9qXJ2v+go="
				if got := server["password"]; got != want {
					t.Fatalf("ss password = %v", got)
				}
			},
		},
		{
			name:     "vless Alice base64 with REALITY",
			uri:      "vless://YXV0bzo3MDcyNjI5ZS05NmMyLTRjMzAtOTEyOC0wOWNiYjhjZTJhNGNAOTEuMTAzLjEyMi4xNDM6MTQzMzQ?remarks=%F0%9F%87%AD%F0%9F%87%B0Alice%20HK(DNS%E8%A7%A3%E9%94%81)%20REALITY&tls=1&peer=download.furry.luxe&xtls=2&pbk=jdmfgRsRAMxoKM_jqgzssx6Hd7GKauIdgOoj0PjTaQk&sid=f192a50c",
			protocol: "vless",
			address:  "91.103.122.143",
			port:     14334,
			check: func(t *testing.T, settings, stream map[string]any) {
				user := firstUser(t, settings, "vnext")
				if got := user["id"]; got != "7072629e-96c2-4c30-9128-09cbb8ce2a4c" {
					t.Fatalf("vless id = %v", got)
				}
				if got := stream["security"]; got != "reality" {
					t.Fatalf("vless security = %v", got)
				}
				reality, ok := stream["realitySettings"].(map[string]any)
				if !ok {
					t.Fatalf("missing realitySettings: %v", stream)
				}
				if got := reality["serverName"]; got != "download.furry.luxe" {
					t.Fatalf("reality serverName = %v", got)
				}
				if got := reality["publicKey"]; got != "jdmfgRsRAMxoKM_jqgzssx6Hd7GKauIdgOoj0PjTaQk" {
					t.Fatalf("reality publicKey = %v", got)
				}
				if got := reality["shortId"]; got != "f192a50c" {
					t.Fatalf("reality shortId = %v", got)
				}
			},
		},
		{
			name:     "vmess Alice base64 with query",
			uri:      "vmess://YXV0bzo3MDcyNjI5ZS05NmMyLTRjMzAtOTEyOC0wOWNiYjhjZTJhNGNAOTEuMTAzLjEyMi4xNDM6MTQzMzU?remarks=%F0%9F%87%AD%F0%9F%87%B0Alice%20HK(DNS%E8%A7%A3%E9%94%81)%20VMess&alterId=0",
			protocol: "vmess",
			address:  "91.103.122.143",
			port:     14335,
			check: func(t *testing.T, settings, stream map[string]any) {
				user := firstUser(t, settings, "vnext")
				if got := user["id"]; got != "7072629e-96c2-4c30-9128-09cbb8ce2a4c" {
					t.Fatalf("vmess id = %v", got)
				}
				if got := numericField(user, "alterId"); got != 0 {
					t.Fatalf("vmess alterId = %v", got)
				}
				if got := user["security"]; got != "auto" {
					t.Fatalf("vmess cipher = %v", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outbound, err := buildOutbound(tc.uri)
			if err != nil {
				t.Fatalf("buildOutbound: %v", err)
			}
			if got := outbound["protocol"]; got != tc.protocol {
				t.Fatalf("protocol = %v want %v", got, tc.protocol)
			}
			settings := outbound["settings"].(map[string]any)
			stream, _ := outbound["streamSettings"].(map[string]any)
			address, port := firstEndpoint(t, tc.protocol, settings)
			if address != tc.address {
				t.Fatalf("address = %v want %v", address, tc.address)
			}
			if port != tc.port {
				t.Fatalf("port = %v want %v", port, tc.port)
			}
			tc.check(t, settings, stream)

			if _, err := json.Marshal(outbound); err != nil {
				t.Fatalf("marshal outbound: %v", err)
			}
		})
	}
}

func firstServer(t *testing.T, settings map[string]any) map[string]any {
	t.Helper()
	servers, ok := settings["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Fatalf("missing servers: %v", settings)
	}
	return servers[0].(map[string]any)
}

func firstUser(t *testing.T, settings map[string]any, key string) map[string]any {
	t.Helper()
	nodes, ok := settings[key].([]any)
	if !ok || len(nodes) == 0 {
		t.Fatalf("missing %s: %v", key, settings)
	}
	users := nodes[0].(map[string]any)["users"].([]any)
	return users[0].(map[string]any)
}

func firstEndpoint(t *testing.T, protocol string, settings map[string]any) (string, int) {
	t.Helper()
	switch protocol {
	case "vmess", "vless":
		node := settings["vnext"].([]any)[0].(map[string]any)
		return node["address"].(string), numericField(node, "port")
	default:
		server := firstServer(t, settings)
		return server["address"].(string), numericField(server, "port")
	}
}

func numericField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
