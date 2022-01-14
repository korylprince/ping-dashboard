package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/korylprince/ipscan/ping"
	"github.com/korylprince/ipscan/resolve"
	"golang.org/x/sync/errgroup"
)

type conn struct {
	*websocket.Conn
	mu *sync.Mutex
}

type Resolve struct {
	Hostname string
	IPs      []net.IP
	Error    error
}

func (r *Resolve) MarshalJSON() ([]byte, error) {
	type resolve struct {
		Type     string   `json:"t"`
		Hostname string   `json:"h"`
		IPs      []string `json:"i,omitempty"`
		Error    string   `json:"e,omitempty"`
	}

	res := &resolve{Type: "r", Hostname: r.Hostname}

	if len(r.IPs) > 0 {
		ips := make([]string, 0, len(r.IPs))
		for _, ip := range r.IPs {
			ips = append(ips, ip.String())
		}
		res.IPs = ips
	}

	if r.Error != nil {
		res.Error = r.Error.Error()
	}

	return json.Marshal(res)
}

type Ping struct {
	*ping.Ping
	Error error
}

func (p *Ping) MarshalJSON() ([]byte, error) {
	type ping struct {
		Type    string `json:"t"`
		IP      string `json:"i"`
		Latency int64  `json:"l"`
		Error   string `json:"e,omitempty"`
	}

	pin := &ping{Type: "p", IP: p.IP.String()}

	if p.Error != nil {
		pin.Error = p.Error.Error()
	}

	if p.Ping != nil && p.Ping.RecvTime != nil {
		pin.Latency = (*p.Ping.RecvTime).Sub(p.SentTime).Microseconds()
	} else if p.Error == nil {
		pin.Error = "no response"
	}

	return json.Marshal(pin)
}

type Service struct {
	Config   *Config
	Resolver *resolve.Service
	Pinger   *ping.Service
	token    string
}

func NewService(config *Config, resolver *resolve.Service, pinger *ping.Service) (*Service, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("could not generate token: %w", err)
	}
	return &Service{Config: config, Resolver: resolver, Pinger: pinger, token: base64.RawURLEncoding.EncodeToString(token)}, nil
}

func (s *Service) resolver(c *conn, hosts <-chan string, ips chan<- net.IP) error {
	for h := range hosts {
		is, err := s.Resolver.LookupIP(h)
		for _, ip := range is {
			ips <- ip
		}

		c.mu.Lock()
		if err := c.WriteJSON(&Resolve{Hostname: h, IPs: is, Error: err}); err != nil {
			c.mu.Unlock()
			return fmt.Errorf("could not write resolved message: %w", err)
		}
		c.mu.Unlock()
	}
	return nil
}

func (s *Service) pinger(c *conn, ips <-chan net.IP) error {
	for ip := range ips {
		p, err := s.Pinger.Ping(ip)

		c.mu.Lock()
		if err := c.WriteJSON(&Ping{Ping: p, Error: err}); err != nil {
			c.mu.Unlock()
			return fmt.Errorf("could not write pinged message: %w", err)
		}
		c.mu.Unlock()
	}
	return nil
}

// HandleConn resolves and pings all of the hosts in schema and handles the full converstion with ws
func (s *Service) HandleConn(ws *websocket.Conn, schema Schema) (err error) {
	type auth struct {
		Token string `json:"token"`
	}

	c := &conn{Conn: ws, mu: new(sync.Mutex)}

	// defer closing ws
	defer func() {
		c.mu.Lock()
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		if e := c.WriteJSON(map[string]string{"t": "c", "e": msg}); err != nil {
			if err == nil {
				err = fmt.Errorf("could not write close message: %w", e)
			}
		}
		c.Close()
		c.mu.Unlock()
	}()

	a := new(auth)
	if err = c.ReadJSON(a); err != nil {
		return fmt.Errorf("could not read token: %w", err)
	}

	if subtle.ConstantTimeEq(int32(len(a.Token)), int32(len(s.token))) != 1 ||
		subtle.ConstantTimeCompare([]byte(a.Token), []byte(s.token)) != 1 {
		return fmt.Errorf("could not authenticate: %w", errors.New("invalid token"))
	}

	if err = c.WriteJSON(schema); err != nil {
		return fmt.Errorf("could not write schema message: %w", err)
	}

	hosts := make(chan string)
	ips := make(chan net.IP)

	wg1 := new(errgroup.Group)
	for i := 0; i < s.Config.Resolvers; i++ {
		wg1.Go(func() error {
			return s.resolver(c, hosts, ips)
		})
	}

	wg2 := new(errgroup.Group)
	for i := 0; i < s.Config.Pingers; i++ {
		wg2.Go(func() error {
			return s.pinger(c, ips)
		})
	}

	for _, hs := range schema {
		for _, h := range hs.Hosts {
			hosts <- h
		}
	}

	close(hosts)
	if err := wg1.Wait(); err != nil {
		close(ips)
		return fmt.Errorf("could not resolve hosts: %w", err)
	}

	close(ips)
	if err := wg2.Wait(); err != nil {
		return fmt.Errorf("could not ping hosts: %w", err)
	}

	return nil
}
