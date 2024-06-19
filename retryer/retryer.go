package retryer

import (
	"context"
	"fmt"
	"time"

	"github.com/quietpleasure/consul"
)

type ServiceConfig struct {
	Name string
	Host string
	Port int
	Tags []string
}

func MakeRegistryAndRegisterService(ctx context.Context, instanceID string, cfgService *ServiceConfig, cfgConsul *consul.Config) (*consul.Registry, error) {
	if cfgConsul == nil {
		cfgConsul = consul.DefaultConfig()
	}
	registry, err := consul.NewRegistry(cfgConsul)
	if err != nil {
		return nil, err
	}
	if cfgService == nil {
		return nil, fmt.Errorf("service configuration not defined")
	}
	if err := registry.Register(cfgService.Name, instanceID, cfgService.Host, cfgService.Port, cfgService.Tags); err != nil {
		return nil, err
	}

	return registry, nil
}

type FuncExecutor func(ctx context.Context, instanceID string, cfgService *ServiceConfig, cfgConsul *consul.Config) (*consul.Registry, error)

type Feedback struct {
	Error   error
	Message string
}

func Retry(function FuncExecutor, feedback chan Feedback, maxAttempts ...int) FuncExecutor {
	var max int
	if len(maxAttempts) > 0 {
		max = maxAttempts[0]
	}
	return func(ctx context.Context, instanceID string, cfgService *ServiceConfig, cfgConsul *consul.Config) (*consul.Registry, error) {
		attempt := 1
		for {
			reg, err := function(ctx, instanceID, cfgService, cfgConsul)
			if err == nil {
				feedback <- Feedback{
					Message: fmt.Sprintf("retry attempt %d successful", attempt),
				}
				return reg, nil
			}
			if attempt == max && max != 0 {
				feedback <- Feedback{
					Message: "all attempts used",
				}
				return reg, err
			}
			delay := time.Second << uint(attempt)
			feedback <- Feedback{
				Error:   err,
				Message: fmt.Sprintf("retry attempt %d failed repeat after %s", attempt, delay),
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
}
