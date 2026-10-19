package consumer

// Option configures consumer options.
type Option func(*options)

type options struct {
	startOffset int64
}

// WithStartOffset configures the consumer's start offset (-2 for FirstOffset, -1 for LastOffset).
func WithStartOffset(offset int64) Option {
	return func(o *options) {
		o.startOffset = offset
	}
}
