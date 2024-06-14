package consul

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
)

var ErrServicesNotFound error = fmt.Errorf("no service addresses found")

// Registry defines a Consul-based service regisry.
type Registry struct {
	client *api.Client
}

// NewRegistry creates a new Consul-based service registry instance.
func NewRegistry(consulAddr string) (*Registry, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Registry{client: client}, nil
}

// Register creates a service record in the registry and return instanceID
func (r *Registry) Register(serviceName, serviceHostPort string) (string, error) {
	parts := strings.Split(serviceHostPort, ":")
	if len(parts) != 2 {
		return "", errors.New("hostPort must be in a form of <host>:<port>, example: localhost:8081")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	instanceID := fmt.Sprintf("%s-%s", serviceName, uuid.New())
	if err := r.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		Address: parts[0],
		ID:      instanceID,
		Name:    serviceName,
		Port:    port,
		Check:   &api.AgentServiceCheck{CheckID: instanceID, TTL: "5s"},
	}); err != nil {
		return "", err
	}
	return instanceID, nil
}

// Deregister removes a service record from the registry.
func (r *Registry) Deregister(instanceID string, _ string) error {
	return r.client.Agent().ServiceDeregister(instanceID)
}

// ServiceAddresses returns the list of addresses of active instances of the given service.
func (r *Registry) ServiceAddresses(serviceName string) ([]string, error) {
	entries, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	} else if len(entries) == 0 {
		return nil, ErrServicesNotFound
	}
	var res []string
	for _, e := range entries {
		res = append(res, fmt.Sprintf("%s:%d", e.Service.Address, e.Service.Port))
	}
	return res, nil
}

// ReportHealthyState is a push mechanism for reporting healthy state to the registry.
func (r *Registry) ReportHealthyState(instanceID string, _ string) error {
	return r.client.Agent().PassTTL(instanceID, "")
}
