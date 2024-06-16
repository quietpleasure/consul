package consul

import (
	"fmt"
	"net/url"

	_ "github.com/mbobakov/grpc-consul-resolver" // It's important

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

type ResolveConfig struct {
	// Select endpoints only with this tag
	Tag string
	// Return only endpoints which pass all health-checks. Default: false
	Healthy bool
	// Wait time for watch changes. Due this time period endpoints will be force refreshed. Default: inherits agent property
	// as in time.ParseDuration
	Wait string
	// Allow insecure communication with Consul. Default: true
	Insecure bool
	// Sort endpoints by response duration. Can be efficient combine with limit parameter default: "_agent"
	Near string
	// Limit number of endpoints for the service. Default: no limit
	Limit int
	// Http-client timeout. Default: 60s
	// as in time.ParseDuration
	Timeout string
	// Max backoff time for reconnect to consul. Reconnects will start from 10ms to max-backoff exponentialy with factor 2. Default: 1s
	// as in time.ParseDuration
	MaxBackoff string
	// Consul token
	Token string
	// Consul datacenter to choose. Optional
	DC string
	// Allow stale results from the agent. https://www.consul.io/api/features/consistency.html#stale
	AllowStale bool
	// RequireConsistent forces the read to be fully consistent. This is more expensive but prevents ever performing a stale read.
	RequireConsistent bool
}

// consul://[user:password@]127.0.0.127:8555/my-service?[healthy=]&[wait=]&[near=]&[insecure=]&[limit=]&[tag=]&[token=]
// After a positive answer, it is advisable defer conn.Close()
func (c *Consul) GRPCserviceConnect(serviceName string, cfg *ResolveConfig) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		c.makeTarget(serviceName, cfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy": "%s"}`, roundrobin.Name)),
	)
}

func (c *Consul) makeTarget(serviceName string, cfg *ResolveConfig) string {
	var userpass *url.Userinfo
	if c.config.User != "" && c.config.Pass != "" {
		userpass = url.UserPassword(c.config.User, c.config.Pass)
	}
	args := url.Values{}
	if cfg != nil {
		if cfg.Tag != "" {
			args.Set("tag", cfg.Tag)
		}
		if cfg.Healthy {
			args.Set("healthy", fmt.Sprintf("%v", cfg.Healthy))
		}
		if cfg.Wait != "" {
			args.Set("wait", cfg.Wait)
		}
		if !cfg.Insecure {
			args.Set("insecure", fmt.Sprintf("%v", cfg.Insecure))
		}
		if cfg.Near != "" {
			args.Set("near", cfg.Near)
		}
	}
	u := url.URL{
		Scheme:   "consul",
		Host:     fmt.Sprintf("%s:%d", c.config.Addr, c.config.Port),
		Path:     serviceName,
		User:     userpass,
		RawQuery: args.Encode(),
	}

	return u.String()
}
