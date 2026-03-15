package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/singhprasan/my-api-gateway/internal/config"
)

type route struct {
	path    string
	rewrite string
	timeout time.Duration
	proxy   *httputil.ReverseProxy
}

type Proxy struct {
	routes []route
}

func New(routes []config.RouteConfig) *Proxy {
	sorted := make([]config.RouteConfig, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Path) > len(sorted[j].Path)
	})

	p := &Proxy{}
	for _, r := range sorted {
		target, err := url.Parse(r.Upstream)
		if err != nil {
			fmt.Printf("invalid upstream URL %s: %v\n", r.Upstream, err)
			continue
		}

		rewrite := r.Rewrite
		matchPath := r.Path

		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = target.Host

				suffix := strings.TrimPrefix(req.URL.Path, matchPath)
				req.URL.Path = joinPath(rewrite, suffix)
				req.URL.RawPath = ""
			},
		}

		p.routes = append(p.routes, route{
			path:    r.Path,
			rewrite: r.Rewrite,
			timeout: r.Timeout.Std(),
			proxy:   rp,
		})
	}

	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rt := range p.routes {
		if strings.HasPrefix(r.URL.Path, rt.path) {
			fmt.Printf("%s %s -> upstream (route: %s)\n", r.Method, r.URL.Path, rt.path)

			ctx, cancel := context.WithTimeout(r.Context(), rt.timeout)
			defer cancel()

			rt.proxy.ServeHTTP(w, r.WithContext(ctx))
			return
		}
	}

	fmt.Printf("%s %s -> no matching route\n", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func joinPath(base, suffix string) string {
	if suffix == "" {
		return base
	}
	if strings.HasSuffix(base, "/") && strings.HasPrefix(suffix, "/") {
		return base + suffix[1:]
	}
	if !strings.HasSuffix(base, "/") && !strings.HasPrefix(suffix, "/") {
		return base + "/" + suffix
	}
	return base + suffix
}
