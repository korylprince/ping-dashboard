package main

import "time"

type Config struct {
	HostsPath string        `required:"true"`
	Pingers   int           `default:"0"`
	Resolvers int           `default:"0"`
	QueueSize int           `default:"1024"`
	Timeout   time.Duration `default:"1s"`

	Username        string        `default:"admin"`
	Password        string        `required:"true"`
	SessionDuration time.Duration `default:"1h"`

	ProxyHeaders bool   `default:"false"`
	ListenAddr   string `default:":80"`
}
