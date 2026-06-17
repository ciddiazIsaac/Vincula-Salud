//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"testing"
	"time"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestClinicalRecordIntegration(t *testing.T) {
	// Cargar certificados mTLS
	cert, err := tls.LoadX509KeyPair("../../certs/client.crt", "../../certs/client.key")
	if err != nil {
		t.Fatalf("No se pudo cargar el certificado del cliente: %v", err)
	}

	caCert, err := os.ReadFile("../../certs/ca.crt")
	if err != nil {
		t.Fatalf("No se pudo leer el certificado CA: %v", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: true, // Para pruebas locales con localhost
	}

	creds := credentials.NewTLS(tlsConfig)

	// Conectar al servidor gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		t.Fatalf("Error conectando al servidor: %v", err)
	}
	defer conn.Close()

	client := clinicalv1.NewClinicalRecordServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	patientRun := "12345678-9"

	// Prueba 1: RecordClinicalEvent
	eventResp, err := client.RecordClinicalEvent(ctx, &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       patientRun,
		EventType:        "Consulta Integración",
		EventDataJson:    []byte(`{"diagnostico": "Prueba de integración exitosa"}`),
		AuthorCredential: "DR-TEST",
	})

	if err != nil {
		t.Fatalf("Fallo al registrar evento: %v", err)
	}
	t.Logf("Evento registrado correctamente con ID: %s", eventResp.EventId)

	// Prueba 2: GetPatientSummary
	_, err = client.GetPatientSummary(ctx, &clinicalv1.GetPatientSummaryRequest{
		PatientRun:          patientRun,
		RequesterHospitalId: "HOSP-001",
	})

	if err != nil {
		t.Logf("GetPatientSummary devolvió error: %v", err)
	} else {
		t.Logf("GetPatientSummary exitoso para %s", patientRun)
	}
}
