package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type pingServiceServer interface {
	Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
}

type pingServiceImpl struct{}

func (s *pingServiceImpl) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

var pingServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.TestService",
	HandlerType: (*pingServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(emptypb.Empty)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*pingServiceImpl).Ping(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/test.TestService/Ping",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(*pingServiceImpl).Ping(ctx, req.(*emptypb.Empty))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
}

func TestServer_Start_EnforcesDefaultInterceptorChain(t *testing.T) {
	srv, err := New(&Config{
		Address:           "127.0.0.1:0",
		EnableHealthCheck: true,
	})
	if err != nil {
		t.Fatalf("new server error = %v", err)
	}
	srv.RegisterService(&pingServiceDesc, &pingServiceImpl{})

	if err := srv.Start(); err != nil {
		t.Fatalf("start server error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(stopCtx)
	})

	conn, err := grpc.NewClient(srv.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("new client error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = conn.Invoke(ctx, "/test.TestService/Ping", &emptypb.Empty{}, &emptypb.Empty{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated from auth interceptor, got %v (err=%v)", status.Code(err), err)
	}
}

func TestServer_Start_SetsServiceSpecificHealthStatus(t *testing.T) {
	srv, err := New(&Config{
		Address:           "127.0.0.1:0",
		EnableHealthCheck: true,
	})
	if err != nil {
		t.Fatalf("new server error = %v", err)
	}
	srv.RegisterService(&pingServiceDesc, &pingServiceImpl{})

	if err := srv.Start(); err != nil {
		t.Fatalf("start server error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(stopCtx)
	})

	conn, err := grpc.NewClient(srv.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("new client error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	serviceResp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "test.TestService"})
	if err != nil {
		t.Fatalf("service health check error = %v", err)
	}
	if serviceResp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("expected service SERVING, got %v", serviceResp.GetStatus())
	}

	globalResp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("global health check error = %v", err)
	}
	if globalResp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("expected global SERVING, got %v", globalResp.GetStatus())
	}
}

func TestServer_Stop_TimeoutMarksStopped(t *testing.T) {
	srv, err := New(&Config{
		Address: "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("new server error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("start server error = %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := srv.Stop(stopCtx); err == nil {
		t.Fatalf("expected timeout-forced stop error")
	}
	if srv.IsRunning() {
		t.Fatalf("server should be marked stopped after forced stop")
	}
}
