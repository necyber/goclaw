package config

import (
	"fmt"

	grpcpkg "github.com/goclaw/goclaw/pkg/grpc"
)

// ToGRPCConfig converts config.GRPCConfig to pkg/grpc.Config
func (g *GRPCConfig) ToGRPCConfig() *grpcpkg.Config {
	cfg := &grpcpkg.Config{
		Address:           fmt.Sprintf(":%d", g.Port),
		MaxConnections:    g.MaxConnections,
		MaxRecvMsgSize:    g.MaxRecvMsgSize,
		MaxSendMsgSize:    g.MaxSendMsgSize,
		EnableReflection:  g.EnableReflection,
		EnableHealthCheck: g.EnableHealthCheck,
	}

	// Convert TLS config
	if g.TLS.Enabled {
		cfg.TLS = &grpcpkg.TLSConfig{
			Enabled:    g.TLS.Enabled,
			CertFile:   g.TLS.CertFile,
			KeyFile:    g.TLS.KeyFile,
			CAFile:     g.TLS.CAFile,
			ClientAuth: g.TLS.ClientAuth,
		}
	}

	// Convert Keepalive config
	cfg.Keepalive = &grpcpkg.KeepaliveConfig{
		MaxIdleSeconds:      g.Keepalive.MaxIdleSeconds,
		MaxAgeSeconds:       g.Keepalive.MaxAgeSeconds,
		MaxAgeGraceSeconds:  g.Keepalive.MaxAgeGraceSeconds,
		TimeSeconds:         g.Keepalive.TimeSeconds,
		TimeoutSeconds:      g.Keepalive.TimeoutSeconds,
		MinTimeSeconds:      g.Keepalive.MinTimeSeconds,
		PermitWithoutStream: g.Keepalive.PermitWithoutStream,
	}

	return cfg
}
