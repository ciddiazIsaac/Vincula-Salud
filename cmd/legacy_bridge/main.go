package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	// Cargar certificado del cliente
	cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}
	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := clinicalv1.NewClinicalRecordServiceClient(conn)

	// Resto igual que antes
	f, err := os.Open("data/legacy_patients.csv")
	if err != nil {
		log.Fatal(err) //nolint:gocritic // defer will not run on startup failure, which is acceptable
	}
	defer f.Close()

	r := csv.NewReader(f)
	_, _ = r.Read() // encabezados
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
				log.Printf("error: %v", err)
			}
		}
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
			log.Printf("error: %v", err)
		}
	}
	log.Println("Legacy bridge con mTLS completado")
}
