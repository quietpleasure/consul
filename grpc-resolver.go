package consul

import (
	"fmt"
	"net/url"
	"time"

	_ "github.com/mbobakov/grpc-consul-resolver" // It's important

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

// Function for passing connection parameters
type Option func(option *options) error

type options struct {
	limit             *int
	tag               *string
	healthy           *string
	wait              *string
	insecure          *string
	near              *string
	timeout           *string
	maxbackoff        *string
	token             *string
	dc                *string
	allowstale        *bool
	requireconsistent *bool
}

// consul://[user:password@]127.0.0.127:8555/my-service?[healthy=]&[wait=]&[near=]&[insecure=]&[limit=]&[tag=]&[token=]
// After a positive answer, it is advisable defer conn.Close()
func (r *Registry) ResolveServiceConnectGRPC(serviceName string, opts ...Option) (*grpc.ClientConn, error) {
	target, err := r.makeTarget(serviceName, opts...)
	if err != nil {
		return nil, fmt.Errorf("decode options: %w", err)
	}
	return grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy": "%s"}`, roundrobin.Name)),
	)
}

func (r *Registry) makeTarget(serviceName string, opts ...Option) (string, error) {
	var userpass *url.Userinfo
	if r.config.User != "" && r.config.Pass != "" {
		userpass = url.UserPassword(r.config.User, r.config.Pass)
	}

	var opt options
	for _, option := range opts {
		if err := option(&opt); err != nil {
			return "", err
		}
	}

	args := url.Values{}
	if opt.tag != nil {
		args.Set("tag", *opt.tag)
	}
	if opt.healthy != nil {
		args.Set("healthy", *opt.healthy)
	}
	if opt.wait != nil {
		args.Set("wait", *opt.wait)
	}
	if opt.insecure != nil {
		args.Set("insecure", *opt.insecure)
	}
	if opt.near != nil {
		args.Set("near", *opt.near)
	}
	if opt.limit != nil {
		args.Set("limit", fmt.Sprintf("%v", *opt.limit))
	}
	if opt.timeout != nil {
		args.Set("timeout", *opt.timeout)
	}
	if opt.maxbackoff != nil {
		args.Set("max-backoff", *opt.maxbackoff)
	}
	if opt.token != nil {
		args.Set("token", *opt.token)
	}
	if opt.dc != nil {
		args.Set("dc", *opt.dc)
	}
	if opt.allowstale != nil {
		args.Set("allow-stale", fmt.Sprintf("%v", *opt.allowstale))
	}
	if opt.requireconsistent != nil {
		args.Set("require-consistent", fmt.Sprintf("%v", *opt.requireconsistent))
	}
	u := url.URL{
		Scheme:   SELF_NAME,
		Host:     fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
		Path:     serviceName,
		User:     userpass,
		RawQuery: args.Encode(),
	}
	return u.String(), nil
}

// Select endpoints only with this tag
func WithTag(tag string) Option {
	return func(options *options) error {
		if tag != "" {
			options.tag = &tag
		}
		return nil
	}
}

// Return only endpoints which pass all health-checks. Default: false
func WithHealthy(healthy bool) Option {
	return func(options *options) error {
		check := "true"
		if healthy {
			options.healthy = &check
		}
		return nil
	}
}

func WithWait(wait time.Duration) Option {
	return func(options *options) error {
		if wait < 0 {
			return fmt.Errorf("wait time cannot be less than zero")
		}
		if wait != 0 {
			duration := wait.String()
			options.wait = &duration
		}
		return nil
	}
}

// Allow insecure communication with Consul. Default: true
func WithInsecure(insecure bool) Option {
	return func(options *options) error {
		allow := "false"
		if !insecure {
			options.insecure = &allow
		}
		return nil
	}
}

const OPT_NEAR_IP = "_ip"

// Sort endpoints by response duration. Can be efficient combine with limit parameter. Default: "_agent".
// Near  - Specifies a node to sort near based on distance sorting using Network Coordinates. The nearest instance to the specified node will be returned first, and subsequent nodes in the response will be sorted in ascending order of estimated round-trip times. If the node given does not exist, the nodes in the response will be shuffled. If unspecified, the response will be shuffled by default.
// _agent - Returns results nearest the agent servicing the request.
// _ip - Returns results nearest to the node associated with the source IP where the query was executed from. For HTTP the source IP is the remote peer's IP address or the value of the X-Forwarded-For header with the header taking precedence. For DNS the source IP is the remote peer's IP address or the value of the EDNS client IP with the EDNS client IP taking precedence.
func WithNear(near string) Option {
	return func(options *options) error {
		if (near != "" && near != OPT_NEAR_IP) || near == "" {
			return nil
		}
		opt := OPT_NEAR_IP
		options.near = &opt
		return nil
	}
}

// Limit number of endpoints for the service. Default: no limit
func WithLimit(limit int) Option {
	return func(options *options) error {
		if limit < 0 {
			return fmt.Errorf("limit cannot be less than zero")
		}
		if limit != 0 {
			options.limit = &limit
		}
		return nil
	}
}

// Http-client timeout. Default: 60s
func WithTimeout(timeout time.Duration) Option {
	return func(options *options) error {
		if timeout < 0 {
			return fmt.Errorf("timeout cannot be less than zero")
		}
		if timeout != 0 {
			duration := timeout.String()
			options.timeout = &duration
		}
		return nil
	}
}

// Max backoff time for reconnect to consul. Reconnects will start from 10ms to max-backoff exponentialy with factor 2. Default: 1s
func WithMaxBackoff(maxbackoff time.Duration) Option {
	return func(options *options) error {
		if maxbackoff < 0 {
			return fmt.Errorf("maxbackoff cannot be less than zero")
		}
		if maxbackoff != 0 {
			duration := maxbackoff.String()
			options.maxbackoff = &duration
		}
		return nil
	}
}

// Consul token
func WithToken(token string) Option {
	return func(options *options) error {
		if token != "" {
			options.token = &token
		}
		return nil
	}
}

// Consul datacenter to choose. Optional
func WithDC(dc string) Option {
	return func(options *options) error {
		if dc != "" {
			options.dc = &dc
		}
		return nil
	}
}

// Allow stale results from the agent. https://www.consul.io/api/features/consistency.html#stale
func WithAllowStale(stale bool) Option {
	return func(options *options) error {
		options.allowstale = &stale
		return nil
	}
}

// RequireConsistent forces the read to be fully consistent. This is more expensive but prevents ever performing a stale read.
func WithRequireConsistent(require bool) Option {
	return func(options *options) error {
		options.requireconsistent = &require
		return nil
	}
}
