package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "log"
    "net"
    "os"
    "sync/atomic"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    
    "github.com/minsal/vincula/internal/adapters"
    clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
)

var isReady atomic.Bool

func main() {
    ctx := context.Background()
    database := os.Getenv("SPANNER_DATABASE")
    if database == "" {
        database = "projects/vincula-salud-dev/instances/vincula-instance/databases/vincula_db"
    }
    
    store, err := adapters.NewSpannerClinicalStore(ctx, database)
    if err != nil {
        log.Fatalf("Failed to connect to Spanner: %v", err)
    }
    defer store.Close()
    
    isReady.Store(true)
    
    // Cargar certificados mTLS
    cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
    if err != nil {
        log.Fatalf("Failed to load server certs: %v", err)
    }
    
    caCert, err := os.ReadFile("certs/ca.crt")
    if err != nil {
        log.Fatalf("Failed to read CA cert: %v", err)
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
    srv := grpc.NewServer(grpc.Creds(creds))
    
    // Registrar servicios
    clinicalv1.RegisterClinicalRecordServiceServer(srv, store)
    
    // Health check gRPC estándar
    healthSrv := health.NewServer()
    grpc_health_v1.RegisterHealthServer(srv, healthSrv)
    healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
    healthSrv.SetServingStatus("clinical-record", grpc_health_v1.HealthCheckResponse_SERVING)
    
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    log.Println("VINCULA Salud gRPC server with mTLS on :50051")
    log.Println("Health check: grpc_health_v1.Health/Check")
    if err := srv.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
