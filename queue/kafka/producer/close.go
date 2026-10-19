package producer

import "github.com/baby-hero-404/gocore/stderr"

// Close flushes buffered messages and closes the underlying writer.
func (p *impl) Close() error {
	if err := p.writer.Close(); err != nil {
		return stderr.Wrap(stderr.CodeInternal, err, "failed to close kafka producer writer")
	}
	return nil
}
