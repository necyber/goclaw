package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Role represents user roles
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// AuthorizationUnaryInterceptor enforces role-based access control
func AuthorizationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authorization for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(ctx, req)
		}

		// Get user ID from context (set by authentication interceptor)
		userID := ctx.Value("user_id")
		if userID == nil {
			return nil, status.Error(codes.PermissionDenied, "user not authenticated")
		}

		// Check if method requires admin role
		if requiresAdminRole(info.FullMethod) {
			role := getUserRole(userID.(string))
			if role != RoleAdmin {
				return nil, status.Error(codes.PermissionDenied, "admin role required")
			}
		}

		return handler(ctx, req)
	}
}

// AuthorizationStreamInterceptor enforces role-based access control for streams
func AuthorizationStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authorization for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Get user ID from context
		userID := ctx.Value("user_id")
		if userID == nil {
			return status.Error(codes.PermissionDenied, "user not authenticated")
		}

		// Check if method requires admin role
		if requiresAdminRole(info.FullMethod) {
			role := getUserRole(userID.(string))
			if role != RoleAdmin {
				return status.Error(codes.PermissionDenied, "admin role required")
			}
		}

		return handler(srv, ss)
	}
}

// requiresAdminRole checks if a method requires admin role
func requiresAdminRole(method string) bool {
	// Admin-only methods (from AdminService)
	adminMethods := map[string]bool{
		"/goclaw.v1.AdminService/GetEngineStatus":  true,
		"/goclaw.v1.AdminService/UpdateConfig":     true,
		"/goclaw.v1.AdminService/ManageCluster":    true,
		"/goclaw.v1.AdminService/PauseWorkflows":   true,
		"/goclaw.v1.AdminService/ResumeWorkflows":  true,
		"/goclaw.v1.AdminService/PurgeWorkflows":   true,
		"/goclaw.v1.AdminService/GetLaneStats":     true,
		"/goclaw.v1.AdminService/ExportMetrics":    true,
		"/goclaw.v1.AdminService/GetDebugInfo":     true,
	}

	return adminMethods[method]
}

// getUserRole retrieves the role for a user
// In production, this should query a database or cache
func getUserRole(userID string) Role {
	// Simplified role lookup - in production query database
	// For now, return user role for all users
	return RoleUser
}
