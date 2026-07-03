package handler

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type HealthChecker struct {
	backends []string
	status   map[string]bool
	mu       sync.RWMutex
	interval time.Duration
	timeout  time.Duration
	client   *http.Client
	stopCh   chan struct{}
}

func NewHealthChecker(backends []string, interval, timeout time.Duration) *HealthChecker {
	hc := &HealthChecker{
		backends: backends,
		status:   make(map[string]bool),
		interval: interval,
		timeout:  timeout,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		stopCh: make(chan struct{}),
	}
	for _, b := range backends {
		hc.status[b] = false
	}
	return hc
}

func (hc *HealthChecker) Start() {
	hc.checkAll()
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hc.checkAll()
			case <-hc.stopCh:
				return
			}
		}
	}()
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

func (hc *HealthChecker) checkAll() {
	for _, backend := range hc.backends {
		healthy := hc.checkBackend(backend)
		hc.mu.Lock()
		hc.status[backend] = healthy
		hc.mu.Unlock()
	}
}

func (hc *HealthChecker) checkBackend(backend string) bool {
	u, err := url.Parse(backend)
	if err != nil {
		return false
	}
	healthURL := u.Scheme + "://" + u.Host + "/health"
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	resp, err := hc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (hc *HealthChecker) AllHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	for _, healthy := range hc.status {
		if !healthy {
			return false
		}
	}
	return true
}

func (hc *HealthChecker) Status() map[string]bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	cp := make(map[string]bool, len(hc.status))
	for k, v := range hc.status {
		cp[k] = v
	}
	return cp
}

func (hc *HealthChecker) Handle(c *gin.Context) {
	status := hc.Status()
	allUp := true
	upstreams := make(map[string]string, len(status))
	for backend, healthy := range status {
		if healthy {
			upstreams[backend] = "up"
		} else {
			upstreams[backend] = "down"
			allUp = false
		}
	}
	if allUp {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "upstreams": upstreams})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "upstreams": upstreams})
	}
}
