package main

import (
    "context"
    "encoding/csv"
    "encoding/json"
    "io"
    "log"
    "os"
    
    clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    client := clinicalv1.NewClinicalRecordServiceClient(conn)
    
    f, err := os.Open("data/legacy_patients.csv")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    r := csv.NewReader(f)
    r.Read() // encabezados
    for {
        record, err := r.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
        run := record[0]
        alergia := record[2]
        diagnostico := record[3]
        
        // Enviar alergia como evento
        if alergia != "ninguna" {
            data, _ := json.Marshal(map[string]string{"alergia": alergia})
            req := &clinicalv1.RecordClinicalEventRequest{
                PatientRun:       run,
                EventType:        "allergy",
                EventDataJson:    data,
                AuthorCredential: "legacy_bridge",
                EventTimestamp:   timestamppb.Now(),
            }
            _, err := client.RecordClinicalEvent(context.Background(), req)
            if err != nil {
                log.Printf("error enviando alergia para %s: %v", run, err)
            }
        }
        // Enviar diagnóstico
        data, _ := json.Marshal(map[string]string{"diagnostico": diagnostico})
        req := &clinicalv1.RecordClinicalEventRequest{
            PatientRun:       run,
            EventType:        "diagnosis",
            EventDataJson:    data,
            AuthorCredential: "legacy_bridge",
            EventTimestamp:   timestamppb.Now(),
        }
        _, err = client.RecordClinicalEvent(context.Background(), req)
        if err != nil {
            log.Printf("error enviando diagnostico para %s: %v", run, err)
        }
    }
    log.Println("Legacy bridge terminó de enviar datos")
}
