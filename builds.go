package consul

import (
	"context"
	"fmt"
	"log"
	"time"
)

type ServiceConfig struct {
	Name string
	Host string
	Port int
	Tags []string
}

func MakeRegistryAndRegisterService(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config) (*Registry, error) {
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

	return registry, nil
}

type Effector func(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config) (*Registry, error)

func Retry(effector Effector) Effector {
	return func(ctx context.Context, cfgService *ServiceConfig, cfgConsul *Config) (*Registry, error) {
		for attempt := 1; ; attempt++ {
			reg, err := effector(ctx, cfgService, cfgConsul)
			if err == nil {
				return reg, nil
			}
			delay := time.Second << uint(attempt)
			log.Printf("Attempt %d failed; Error: %s; retrying in %v", attempt, err, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
}


