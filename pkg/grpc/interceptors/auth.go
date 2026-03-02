package interceptors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	// AuthorizationKey is the metadata key for authorization token
	AuthorizationKey = "authorization"
	// AuthSecretEnv is the environment variable used for HMAC token validation.
	AuthSecretEnv = "GOCLAW_AUTH_TOKEN_SECRET"
)

// AuthenticationUnaryInterceptor validates authentication tokens
func AuthenticationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(ctx, req)
		}

		// Prefer token auth; fallback to mTLS identity for service-to-service calls.
		if userID, roles, ok := authenticateFromMetadata(ctx); ok {
			ctx = withUserID(ctx, userID)
			ctx = withRoles(ctx, roles)
			return handler(ctx, req)
		}

		if userID, roles, ok := authenticateFromMTLS(ctx); ok {
			ctx = withUserID(ctx, userID)
			ctx = withRoles(ctx, roles)
			return handler(ctx, req)
		}

		return nil, status.Error(codes.Unauthenticated, "missing valid authentication credentials")
	}
}

// AuthenticationStreamInterceptor validates authentication tokens for streams
func AuthenticationStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		if userID, roles, ok := authenticateFromMetadata(ctx); ok {
			ctx = withUserID(ctx, userID)
			ctx = withRoles(ctx, roles)
			// Wrap stream with new context
			wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
			return handler(srv, wrapped)
		}

		if userID, roles, ok := authenticateFromMTLS(ctx); ok {
			ctx = withUserID(ctx, userID)
			ctx = withRoles(ctx, roles)
			// Wrap stream with new context
			wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
			return handler(srv, wrapped)
		}

		return status.Error(codes.Unauthenticated, "missing valid authentication credentials")
	}
}

type tokenClaims struct {
	Subject string   `json:"sub"`
	Roles   []string `json:"roles"`
	Expiry  int64    `json:"exp,omitempty"`
}

func authenticateFromMetadata(ctx context.Context) (string, []Role, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", nil, false
	}

	values := md.Get(AuthorizationKey)
	if len(values) == 0 {
		return "", nil, false
	}
	userID, roles, err := validateToken(values[0])
	if err != nil {
		return "", nil, false
	}
	return userID, roles, true
}

func authenticateFromMTLS(ctx context.Context) (string, []Role, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return "", nil, false
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", nil, false
	}
	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.PeerCertificates) == 0 {
		return "", nil, false
	}

	cert := tlsInfo.State.PeerCertificates[0]
	userID := strings.TrimSpace(cert.Subject.CommonName)
	if userID == "" {
		userID = strings.TrimSpace(cert.Subject.String())
	}
	if userID == "" {
		return "", nil, false
	}

	roles := make([]Role, 0, len(cert.Subject.OrganizationalUnit))
	for _, ou := range cert.Subject.OrganizationalUnit {
		switch strings.ToLower(strings.TrimSpace(ou)) {
		case string(RoleAdmin):
			roles = append(roles, RoleAdmin)
		case string(RoleUser):
			roles = append(roles, RoleUser)
		}
	}
	if len(roles) == 0 {
		roles = append(roles, RoleUser)
	}
	return userID, roles, true
}

// validateToken validates a signed bearer token:
// "Bearer <base64url(payload)>.<base64url(signature)>"
func validateToken(header string) (string, []Role, error) {
	token, err := parseBearerToken(header)
	if err != nil {
		return "", nil, err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid token format")
	}

	payloadSegment := parts[0]
	signatureSegment := parts[1]
	if payloadSegment == "" || signatureSegment == "" {
		return "", nil, fmt.Errorf("invalid token parts")
	}

	if !verifyTokenSignature(payloadSegment, signatureSegment) {
		return "", nil, fmt.Errorf("invalid token signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadSegment)
	if err != nil {
		return "", nil, fmt.Errorf("invalid token payload encoding")
	}

	var claims tokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", nil, fmt.Errorf("invalid token payload")
	}

	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		return "", nil, fmt.Errorf("missing subject claim")
	}

	if claims.Expiry > 0 && time.Now().Unix() > claims.Expiry {
		return "", nil, fmt.Errorf("token expired")
	}

	roles := normalizeRoles(claims.Roles)
	return subject, roles, nil
}

func parseBearerToken(header string) (string, error) {
	value := strings.TrimSpace(header)
	if value == "" {
		return "", fmt.Errorf("empty token")
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization header must use Bearer scheme")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}

func verifyTokenSignature(payloadSegment string, signatureSegment string) bool {
	decodedSig, err := base64.RawURLEncoding.DecodeString(signatureSegment)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, tokenSecret())
	_, _ = mac.Write([]byte(payloadSegment))
	expected := mac.Sum(nil)
	return hmac.Equal(decodedSig, expected)
}

func tokenSecret() []byte {
	secret := strings.TrimSpace(os.Getenv(AuthSecretEnv))
	if secret == "" {
		secret = "goclaw-dev-token-secret"
	}
	return []byte(secret)
}

func normalizeRoles(roles []string) []Role {
	normalized := make([]Role, 0, len(roles))
	seen := map[Role]struct{}{}

	for _, role := range roles {
		r := Role(strings.ToLower(strings.TrimSpace(role)))
		if r != RoleAdmin && r != RoleUser {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		normalized = append(normalized, r)
	}
	if len(normalized) == 0 {
		normalized = append(normalized, RoleUser)
	}
	return normalized
}
