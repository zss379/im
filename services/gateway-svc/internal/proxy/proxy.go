package proxy

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type RouteProxy struct {
	Prefix  string
	Backend string
	proxy   *http.ReverseProxy
}

type Manager struct {
	proxies []*RouteProxy
	mu      sync.RWMutex
}

func NewManager(routes []struct {
	Prefix  string
	Backend string
	Timeout time.Duration
}) *Manager {
	m := &Manager{}
	for _, r := range routes {
		target, err := url.Parse(r.Backend)
		if err != nil {
			log.Fatal().Err(err).Str("backend", r.Backend).Msg("invalid backend URL")
			continue
		}

		transport := &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		}

		rp := &http.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				if _, ok := req.Header["X-Forwarded-For"]; !ok {
					req.Header.Set("X-Forwarded-For", req.RemoteAddr)
				}
			},
			Transport: transport,
			ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
				log.Warn().Err(err).Str("path", req.URL.Path).Msg("proxy error")
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte(`{"code":502,"message":"upstream unavailable"}`))
			},
		}

		m.proxies = append(m.proxies, &RouteProxy{
			Prefix:  r.Prefix,
			Backend: r.Backend,
			proxy:   rp,
		})
	}
	return m
}

func (m *Manager) Find(path string) *RouteProxy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.proxies {
		if strings.HasPrefix(path, p.Prefix) {
			return p
		}
	}
	return nil
}

func (r *RouteProxy) Proxy() *http.ReverseProxy {
	return r.proxy
}

func (m *Manager) Backends() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	backends := make([]string, len(m.proxies))
	for i, p := range m.proxies {
		backends[i] = p.Backend
	}
	return backends
}
