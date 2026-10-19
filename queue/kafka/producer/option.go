package producer

import "github.com/segmentio/kafka-go"

// Option configures producer options.
type Option func(*options)

type options struct {
	balancer kafka.Balancer
	async    bool
}

// WithBalancer overrides the default partition balancer (default: LeastBytes).
func WithBalancer(balancer kafka.Balancer) Option {
	return func(o *options) {
		if balancer != nil {
			o.balancer = balancer
		}
	}
}

// WithAsync enables asynchronous background writes for higher throughput.
func WithAsync(async bool) Option {
	return func(o *options) {
		o.async = async
	}
}
