package middleware

import "context"

// RouteInfo carries route-match details through the request context.
type RouteInfo struct {
	MatchedRoute string
}

type routeContextKey struct{}

// WithRouteInfo returns a new context carrying the given RouteInfo.
func WithRouteInfo(ctx context.Context, ri *RouteInfo) context.Context {
	return context.WithValue(ctx, routeContextKey{}, ri)
}

// GetRouteInfo retrieves the RouteInfo from ctx, or nil if absent.
func GetRouteInfo(ctx context.Context) *RouteInfo {
	ri, _ := ctx.Value(routeContextKey{}).(*RouteInfo)
	return ri
}

// APIKeyInfo carries authenticated API key details through the request context.
type APIKeyInfo struct {
	KeyPrefix string
}

type apiKeyContextKey struct{}

func WithAPIKeyInfo(ctx context.Context, info *APIKeyInfo) context.Context {
	return context.WithValue(ctx, apiKeyContextKey{}, info)
}

func GetAPIKeyInfo(ctx context.Context) *APIKeyInfo {
	info, _ := ctx.Value(apiKeyContextKey{}).(*APIKeyInfo)
	return info
}
