package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"time"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	log.Println("Iniciando cliente de prueba VINCULA Salud...")

	// 1. Cargar certificados mTLS
	cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatalf("Error cargando certificado de cliente: %v\n(Asegúrate de haber generado los certificados en la carpeta 'certs/')", err)
	}

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("Error leyendo CA: %v", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}

	creds := credentials.NewTLS(tlsConfig)

	// 2. Conectar al servidor gRPC
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Error conectando al servidor: %v", err)
	}
	defer conn.Close()

	client := clinicalv1.NewClinicalRecordServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	patientRun := "12345678-9"

	// 3. Registrar un evento clínico (Alergia)
	log.Println("Registrando evento clínico (Alergia)...")
	eventReq := &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       patientRun,
		EventType:        "allergy",
		EventDataJson:    []byte(`{"alergia": "Penicilina", "severidad": "alta"}`),
		AuthorCredential: "DR-EJEMPLO",
		EventTimestamp:   timestamppb.Now(),
	}

	eventResp, err := client.RecordClinicalEvent(ctx, eventReq)
	if err != nil {
		log.Fatalf("Fallo al registrar evento: %v", err)
	}
	log.Printf("✅ Evento registrado exitosamente! ID: %s", eventResp.EventId)

	// 4. Obtener el resumen clínico del paciente
	log.Println("Consultando resumen del paciente...")
	summaryReq := &clinicalv1.GetPatientSummaryRequest{
		PatientRun:          patientRun,
		RequesterHospitalId: "HOSP-TEST",
	}

	summaryResp, err := client.GetPatientSummary(ctx, summaryReq)
	if err != nil {
		log.Fatalf("Fallo al obtener resumen: %v", err)
	}

	log.Printf("✅ Resumen obtenido para RUN: %s", summaryResp.PatientRun)
	log.Printf("   Alergias activas: %v", summaryResp.ActiveAllergies)
	log.Printf("   Diagnósticos activos: %v", summaryResp.ActiveDiagnoses)
	log.Printf("   Medicamentos activos: %v", summaryResp.ActiveMedications)
	log.Printf("   Última actualización: %s", summaryResp.LastUpdate.AsTime().Format(time.RFC3339))
}
