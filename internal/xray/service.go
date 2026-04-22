package xray

import (
	"fmt"
	"strings"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type Instance struct {
	instance *core.Instance
}

func StartProxy(proxyURI string, listenHost string, listenPort int, debug bool) (*Instance, error) {
	configJSON, err := BuildConfigJSON(proxyURI, listenHost, listenPort, debug)
	if err != nil {
		return nil, err
	}

	config, err := serial.LoadJSONConfig(strings.NewReader(configJSON))
	if err != nil {
		return nil, fmt.Errorf("load Xray config: %w", err)
	}

	instance, err := core.New(config)
	if err != nil {
		return nil, fmt.Errorf("create Xray instance: %w", err)
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return nil, fmt.Errorf("start Xray instance: %w", err)
	}

	return &Instance{instance: instance}, nil
}

func (x *Instance) Close() error {
	if x == nil || x.instance == nil {
		return nil
	}
	return x.instance.Close()
}
