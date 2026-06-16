package main

import (
    "context"
    "log"
    "net"
    "os"
    
    "github.com/minsal/vincula/internal/adapters"
    clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
    "google.golang.org/grpc"
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
    
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    srv := grpc.NewServer()
    clinicalv1.RegisterClinicalRecordServiceServer(srv, store)
    log.Println("VINCULA Salud gRPC server with Spanner on :50051")
    if err := srv.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
