package interceptors

import "context"

type contextKey string

const (
	userIDContextKey    contextKey = "user_id"
	requestIDContextKey contextKey = "request_id"
	rolesContextKey     contextKey = "roles"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func userIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func requestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	return requestID, ok
}

func withRoles(ctx context.Context, roles []Role) context.Context {
	if len(roles) == 0 {
		return context.WithValue(ctx, rolesContextKey, []Role{RoleUser})
	}
	cp := make([]Role, 0, len(roles))
	cp = append(cp, roles...)
	return context.WithValue(ctx, rolesContextKey, cp)
}

func rolesFromContext(ctx context.Context) ([]Role, bool) {
	roles, ok := ctx.Value(rolesContextKey).([]Role)
	if !ok {
		return nil, false
	}
	cp := make([]Role, 0, len(roles))
	cp = append(cp, roles...)
	return cp, true
}
