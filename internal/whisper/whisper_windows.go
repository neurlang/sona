//go:build windows

package whisper

type Context struct{}

func New(modelPath string) (*Context, error) {
	return nil, ErrNotImplemented
}

func (c *Context) Transcribe(samples []float32) (string, error) {
	return "", ErrNotImplemented
}

func (c *Context) Close() {}
