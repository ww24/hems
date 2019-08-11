package dongle

import "go.uber.org/zap"

// Option represents hems client option.
type Option interface {
	// Apply applies client options.
	Apply(*Client)
}

func (d *Client) applyOptions(opts []Option) {
	for _, o := range opts {
		o.Apply(d)
	}
}

// NewRbID returns rbID option.
func NewRbID(v string) RbID {
	return RbID(v)
}

// RbID represents rbID parameter option.
type RbID string

// Apply applies rbID parameter.
func (r RbID) Apply(d *Client) {
	d.rbID = string(r)
}

// NewPwd returns pwd option.
func NewPwd(v string) Pwd {
	return Pwd(v)
}

// Pwd represents pwd parameter option.
type Pwd string

// Apply applies pwd parameter.
func (p Pwd) Apply(d *Client) {
	d.pwd = string(p)
}

// NewLogger returns logger option.
func NewLogger(l *zap.Logger) *Logger {
	return (*Logger)(l)
}

// Logger represents logger parameter option.
type Logger zap.Logger

// Apply applies logger parameter.
func (l *Logger) Apply(d *Client) {
	d.Logger = (*zap.Logger)(l)
}
