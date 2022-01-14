package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/didip/tollbooth/v6/limiter"
	"github.com/gorilla/handlers"
	"github.com/kelseyhightower/envconfig"
	"github.com/korylprince/ipscan/ping"
	"github.com/korylprince/ipscan/resolve"
)

func RunServer() error {
	config := new(Config)
	err := envconfig.Process("", config)
	if err != nil {
		return fmt.Errorf("could not process configuration from environment: %w", err)
	}

	if config.Resolvers == 0 {
		config.Resolvers = runtime.NumCPU() * 4
	}
	if config.Pingers == 0 {
		config.Pingers = runtime.NumCPU() * 2
	}

	resolver := resolve.NewService(config.Resolvers, config.QueueSize)

	pinger, ips, err := ping.NewService(config.Pingers, config.QueueSize, config.Timeout, nil)
	if err != nil {
		return fmt.Errorf("could not start ping service: %w", err)
	}

	is := make([]string, 0, len(ips))
	for _, ip := range ips {
		is = append(is, ip.String())
	}
	log.Println("Listening for ICMP on:", strings.Join(is, ", "))

	svc, err := NewService(config, resolver, pinger)
	if err != nil {
		return fmt.Errorf("could not start service: %w", err)
	}

	authmux := http.NewServeMux()
	distFS, _ := fs.Sub(dist, "ui/dist")
	authmux.Handle("/", http.FileServer(&EmbedFS{http.FS(distFS)}))
	authmux.Handle("/ws", svc.HandlePing())

	mux := http.NewServeMux()
	mux.Handle("/", svc.RequireAuth(authmux))

	lmt := limiter.New(&limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).
		SetMax(float64(config.AuthRateLimit) / 60).
		SetBurst(config.AuthRateLimit).
		SetIPLookups([]string{"RemoteAddr"})

	mux.Handle("/auth", LimitHandler(lmt, svc.AuthHandler()))

	var handler http.Handler = LogHandler(NewLogger(os.Stdout), handlers.CompressHandler(mux))

	// rewrite for x-forwarded-for, etc headers
	if config.ProxyHeaders {
		handler = handlers.ProxyHeaders(handler)
	}

	log.Println("Listening on:", config.ListenAddr)

	return http.ListenAndServe(config.ListenAddr, handler)
}

func main() {
	if err := RunServer(); err != nil {
		log.Println("could not start server:", err)
	}
}
