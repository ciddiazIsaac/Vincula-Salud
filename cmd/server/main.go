package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"github.com/minsal/vincula/internal/adapters"
	"github.com/minsal/vincula/internal/telemetry"
)

var isReady atomic.Bool

func main() {
	ctx := context.Background()

	// Initialize Telemetry
	tp, err := telemetry.Init(ctx, "vincula-salud-clinical")
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				slog.Error("Failed to shutdown tracer provider", "error", err)
			}
		}()
	}

	database := os.Getenv("SPANNER_DATABASE")
	if database == "" {
		database = "projects/vincula-salud-dev/instances/vincula-instance/databases/vincula_db"
	}

	store, err := adapters.NewSpannerClinicalStore(ctx, database)
	if err != nil {
		slog.Error("Failed to connect to Spanner", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	isReady.Store(true)

	// Cargar certificados mTLS
	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		slog.Error("Failed to load server certs", "error", err)
		os.Exit(1)
	}

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		slog.Error("Failed to read CA cert", "error", err)
		os.Exit(1)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	creds := credentials.NewTLS(tlsConfig)
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)

	// Registrar servicios
	clinicalv1.RegisterClinicalRecordServiceServer(srv, store)

	// Registrar métricas gRPC de Prometheus
	grpc_prometheus.Register(srv)

	// Servidor HTTP para métricas de Prometheus
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("Metrics server started", "port", 9090)
		if err := http.ListenAndServe(":9090", nil); err != nil {
			slog.Error("Failed to serve metrics", "error", err)
		}
	}()

	// Health check gRPC estándar
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("clinical-record", grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}
	slog.Info("VINCULA Salud gRPC server started", "port", 50051, "mtls", true)
	slog.Info("Health check enabled", "service", "grpc_health_v1.Health/Check")
	if err := srv.Serve(lis); err != nil {
		slog.Error("Failed to serve", "error", err)
		os.Exit(1)
	}
}
