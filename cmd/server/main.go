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
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    var store clinicalv1.ClinicalRecordServiceServer
    dsn := os.Getenv("SPANNER_DSN")
    if dsn != "" {
        spannerStore, err := adapters.NewSpannerStore(context.Background(), dsn)
        if err != nil {
            log.Fatalf("failed to init spanner store: %v", err)
        }
        defer spannerStore.Close()
        store = spannerStore
        log.Println("Using Spanner storage")
    } else {
        store = adapters.NewInMemoryClinicalStore()
        log.Println("Using In-Memory storage (SPANNER_DSN not set)")
    }

    srv := grpc.NewServer()
    clinicalv1.RegisterClinicalRecordServiceServer(srv, store)
    log.Println("VINCULA Salud gRPC server on :50051")
    if err := srv.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
