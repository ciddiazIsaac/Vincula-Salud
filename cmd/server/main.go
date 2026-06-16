package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "log"
    "net"
    "os"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    
    "github.com/minsal/vincula/internal/adapters"
    clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
)

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
    
    // Cargar certificados del servidor
    cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
    if err != nil {
        log.Fatalf("Failed to load server certs: %v", err)
    }
    
    // Cargar CA para validar clientes
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
    
    clinicalv1.RegisterClinicalRecordServiceServer(srv, store)
    
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    log.Println("VINCULA Salud gRPC server with mTLS on :50051")
    if err := srv.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
