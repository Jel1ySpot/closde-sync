package xray

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

// portProbeAttempts bounds how far StartProxy shifts forward looking for a free
// port before giving up.
const portProbeAttempts = 100

type Instance struct {
	instance *core.Instance
}

// StartProxy launches an embedded Xray HTTP proxy. It begins probing at
// listenPort and shifts forward to the next port whenever one is already in
// use, returning the port it actually bound to so callers can point downstream
// traffic at it.
func StartProxy(proxyURI string, listenHost string, listenPort int, debug bool) (*Instance, int, error) {
	port, err := findAvailablePort(listenHost, listenPort)
	if err != nil {
		return nil, 0, err
	}

	configJSON, err := BuildConfigJSON(proxyURI, listenHost, port, debug)
	if err != nil {
		return nil, 0, err
	}

	config, err := serial.LoadJSONConfig(strings.NewReader(configJSON))
	if err != nil {
		return nil, 0, fmt.Errorf("load Xray config: %w", err)
	}

	instance, err := core.New(config)
	if err != nil {
		return nil, 0, fmt.Errorf("create Xray instance: %w", err)
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return nil, 0, fmt.Errorf("start Xray instance: %w", err)
	}

	return &Instance{instance: instance}, port, nil
}

// findAvailablePort returns the first port at or after startPort on host that
// accepts a TCP listener.
func findAvailablePort(host string, startPort int) (int, error) {
	for port := startPort; port < startPort+portProbeAttempts; port++ {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port in range %d-%d on %s", startPort, startPort+portProbeAttempts-1, host)
}

func (x *Instance) Close() error {
	if x == nil || x.instance == nil {
		return nil
	}
	return x.instance.Close()
}
