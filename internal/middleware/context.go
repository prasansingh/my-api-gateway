package middleware

import "context"

// RouteInfo carries route-match details through the request context.
type RouteInfo struct {
	MatchedRoute string
}

type contextKey struct{}

// WithRouteInfo returns a new context carrying the given RouteInfo.
func WithRouteInfo(ctx context.Context, ri *RouteInfo) context.Context {
	return context.WithValue(ctx, contextKey{}, ri)
}

// GetRouteInfo retrieves the RouteInfo from ctx, or nil if absent.
func GetRouteInfo(ctx context.Context) *RouteInfo {
	ri, _ := ctx.Value(contextKey{}).(*RouteInfo)
	return ri
}
