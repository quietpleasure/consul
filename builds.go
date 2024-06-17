package consul

import (
	"context"
	"fmt"
	"time"
)

type ServiceConfig struct {
	Name string
	Host string
	Port int
	Tags []string
}

func MakeRegistryAndRegisterService(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config, chanErr chan error) (*Registry, error) {
	if cfgConsul == nil {
		cfgConsul = DefaultConfig()
	}
	registry, err := NewRegistry(cfgConsul)
	if err != nil {
		return nil, err
	}
	if cfgService == nil {
		return nil, fmt.Errorf("service configuration not defined")
	}
	if err := registry.Register(cfgService.Name, cfgService.Host, cfgService.Port, cfgService.Tags); err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := registry.ReportHealthyState(); err != nil {
					//отвалился коннект к Консулу, нужно переподключать
					chanErr <- fmt.Errorf("report healthy state: %w", err)
					return
				}
			}
			time.Sleep(time.Second)
		}
	}()
	return registry, nil
}

type Effector func(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config, chanErr chan error) (*Registry, error) 
func Retry(effector Effector) Effector {
	return func(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config, chanErr chan error) (*Registry, error) {
		for attempt := 1; ; attempt++ {
			reg, err := effector(ctx, cfgService, cfgConsul, chanErr)
			if err == nil {
				return reg, nil
			}
			delay := time.Second << uint(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():				
				return nil, ctx.Err()
			}
		}
	}
}