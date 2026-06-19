package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	grpcadapter "github.com/minsal/vincula/internal/adapters/grpc"
	"github.com/minsal/vincula/internal/adapters/storage"
	"github.com/minsal/vincula/internal/adapters/broker"
	"github.com/minsal/vincula/internal/core/usecases"
	"github.com/minsal/vincula/internal/middleware"
	"github.com/minsal/vincula/internal/telemetry"
)

func main() {
	ctx := context.Background()

	// Initialize Telemetry
	tp, err := telemetry.Init(ctx, "vincula-salud-clinical")
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := tp.Shutdown(shutdownCtx); err != nil {
				slog.Error("Failed to shutdown tracer provider", "error", err)
			}
		}()
	}

	database := os.Getenv("SPANNER_DATABASE")
	if database == "" {
		database = "projects/vincula-salud-dev/instances/vincula-instance/databases/vincula_db"
	}

	repo, spannerClient, err := storage.NewSpannerClinicalRepo(ctx, database)
	if err != nil {
		slog.Error("Failed to connect to Spanner", "error", err)
		os.Exit(1) //nolint:gocritic // defer will not run on startup failure, which is acceptable
	}
	defer spannerClient.Close()

	cbRepo := storage.NewCircuitBreakerRepo(repo)

	// Setup Pub/Sub Publisher
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = "vincula-salud-dev"
	}
	pubsubPublisher, err := broker.NewPubSubPublisher(projectID, "clinical-events-topic")
	if err != nil {
		slog.Warn("Failed to initialize PubSub publisher, events will not be published", "error", err)
	} else {
		defer pubsubPublisher.Close()
	}

	useCase := usecases.NewClinicalUseCase(cbRepo, pubsubPublisher)
	clinicalServer := grpcadapter.NewClinicalServer(useCase)

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
	if !caPool.AppendCertsFromPEM(caCert) {
		slog.Error("Failed to parse CA certificate")
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}

	creds := credentials.NewTLS(tlsConfig)

	// Initialize Rate Limiter
	rl := middleware.NewRateLimiter(rate.Limit(10), 20)

	// Build interceptor chain: Auth → JWTAuth → RateLimit → Validation → Audit → Prometheus → OTel
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			middleware.AuthUnaryInterceptor,
			middleware.JWTAuthUnaryInterceptor,
			middleware.RateLimitUnaryInterceptor(rl),
			middleware.ValidationUnaryInterceptor,
			middleware.AuditUnaryInterceptor,
			grpc_prometheus.UnaryServerInterceptor,
		),
		grpc.ChainStreamInterceptor(
			middleware.AuthStreamInterceptor,
			middleware.JWTAuthStreamInterceptor,
			middleware.RateLimitStreamInterceptor(rl),
			grpc_prometheus.StreamServerInterceptor,
		),
	)

	// Registrar servicios
	clinicalv1.RegisterClinicalRecordServiceServer(srv, clinicalServer)

	// Registrar métricas gRPC de Prometheus
	grpc_prometheus.Register(srv)

	// Health check gRPC estándar
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("clinical-record", grpc_health_v1.HealthCheckResponse_SERVING)

	// Servidor HTTP dedicado para métricas de Prometheus (con mux aislado y timeouts)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		slog.Info("Metrics server started", "port", 9090)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

	// Iniciar servidor gRPC
	lis, err := net.Listen("tcp", ":50051") //nolint:gosec // intentional binding to all interfaces in container
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	slog.Info("VINCULA Salud gRPC server started",
		"port", 50051,
		"mtls", true,
		"tls_min_version", "1.3",
		"interceptors", []string{"auth", "jwt_auth", "validation", "audit", "prometheus", "otel"},
	)

	// Graceful shutdown: escuchar señales del sistema
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		slog.Info("Received shutdown signal, draining connections...", "signal", sig.String())

		// Marcar como no-serving para que K8s deje de enviar tráfico
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		healthSrv.SetServingStatus("clinical-record", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		// Drenar conexiones gRPC existentes
		srv.GracefulStop()

		// Apagar servidor de métricas
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Metrics server shutdown error", "error", err)
		}

		slog.Info("Graceful shutdown complete")
	}()

	if err := srv.Serve(lis); err != nil {
		slog.Error("gRPC server stopped", "error", err)
	}
}
