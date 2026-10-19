package consumer

import "github.com/baby-hero-404/gocore/stderr"

// Close closes the underlying reader stream.
func (c *impl) Close() error {
	if err := c.reader.Close(); err != nil {
		return stderr.Wrap(stderr.CodeInternal, err, "failed to close kafka consumer reader")
	}
	return nil
}
