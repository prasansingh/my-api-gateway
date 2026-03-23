package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/singhprasan/my-api-gateway/internal/config"
)

var upstreamHealthGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "gateway_upstream_health",
		Help: "Health status of upstream services (1 = healthy, 0 = unhealthy).",
	},
	[]string{"upstream"},
)

func init() {
	prometheus.MustRegister(upstreamHealthGauge)
}

type UpstreamStatus struct {
	Healthy     bool      `json:"healthy"`
	LastChecked time.Time `json:"last_checked"`
	Latency     string    `json:"latency"`
	LastError   string    `json:"last_error,omitempty"`
}

type Checker struct {
	upstreams []string
	interval  time.Duration
	timeout   time.Duration

	mu     sync.RWMutex
	status map[string]*UpstreamStatus

	stopCh chan struct{}
}

func New(routes []config.RouteConfig, interval, timeout time.Duration) *Checker {
	seen := make(map[string]struct{})
	var upstreams []string
	for _, r := range routes {
		if _, ok := seen[r.Upstream]; !ok {
			seen[r.Upstream] = struct{}{}
			upstreams = append(upstreams, r.Upstream)
		}
	}

	status := make(map[string]*UpstreamStatus, len(upstreams))
	for _, u := range upstreams {
		status[u] = &UpstreamStatus{}
	}

	return &Checker{
		upstreams: upstreams,
		interval:  interval,
		timeout:   timeout,
		status:    status,
		stopCh:    make(chan struct{}),
	}
}

func (c *Checker) Start() {
	c.check()
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.safeCheck()
			case <-c.stopCh:
				slog.Info("health checker stopped")
				return
			}
		}
	}()
}

func (c *Checker) Stop() {
	close(c.stopCh)
}

func (c *Checker) safeCheck() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("health checker panic", "error", r)
		}
	}()
	c.check()
}

func (c *Checker) check() {
	for _, upstream := range c.upstreams {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream+"/healthz", nil)
		if err != nil {
			cancel()
			c.setStatus(upstream, false, time.Since(start), err.Error())
			slog.Warn("health check request error", "upstream", upstream, "error", err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		latency := time.Since(start)
		cancel()
		if err != nil {
			c.setStatus(upstream, false, latency, err.Error())
			slog.Warn("health check failed", "upstream", upstream, "error", err)
			continue
		}
		resp.Body.Close()
		c.setStatus(upstream, true, latency, "")
	}
}

func (c *Checker) setStatus(upstream string, healthy bool, latency time.Duration, lastErr string) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	upstreamHealthGauge.WithLabelValues(upstream).Set(val)

	c.mu.Lock()
	c.status[upstream] = &UpstreamStatus{
		Healthy:     healthy,
		LastChecked: time.Now(),
		Latency:     latency.String(),
		LastError:   lastErr,
	}
	c.mu.Unlock()
}

func (c *Checker) LivezHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}
}

func (c *Checker) ReadyzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		upstreams := make(map[string]*UpstreamStatus, len(c.status))
		allHealthy := true
		for u, s := range c.status {
			upstreams[u] = s
			if !s.Healthy {
				allHealthy = false
			}
		}
		c.mu.RUnlock()

		status := "ready"
		code := http.StatusOK
		if !allHealthy {
			status = "not_ready"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{
			"status":    status,
			"upstreams": upstreams,
		})
	}
}
