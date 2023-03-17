package main

import "time"

// Config configures ping-dashboard
type Config struct {
	HostsPath string        `required:"true"`
	Pingers   int           `default:"0"`
	Resolvers int           `default:"0"`
	QueueSize int           `default:"1024"`
	Timeout   time.Duration `default:"1s"`

	Username        string        `default:"admin"`
	Password        string        `required:"true"`
	AuthRateLimit   int           `default:"3"` // 3 requests per minute
	SessionDuration time.Duration `default:"30m"`

	ProxyHeaders bool   `default:"false"`
	ListenAddr   string `default:":80"`
}
