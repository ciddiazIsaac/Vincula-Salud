//go:build integration

// Package integration contains end-to-end tests that spin up a real gRPC
// server with mTLS and exercise the ClinicalRecordService through a real
// network connection.  They use an in-memory repository so no external
// infrastructure (Spanner, Pub/Sub) is required.
//
// Run with:
//
//	go test -tags=integration -v ./tests/integration/...
package integration

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	grpcadapter "github.com/minsal/vincula/internal/adapters/grpc"
	"github.com/minsal/vincula/internal/adapters/storage"
	"github.com/minsal/vincula/internal/core/usecases"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// Test-scoped TLS helpers
// ---------------------------------------------------------------------------

// tlsBundle holds a CA, server, and client certificate/key pair generated
// for a single test run.
type tlsBundle struct {
	caPool     *x509.CertPool
	serverCert tls.Certificate
	clientCert tls.Certificate
}

// newTLSBundle creates a self-signed CA plus server & client certs valid for
// localhost.  All material lives in memory — nothing touches disk.
func newTLSBundle(t *testing.T) *tlsBundle {
	t.Helper()

	// --- CA ---
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"VINCULA Test CA"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	// helper: issue a leaf certificate signed by the CA.
	issueCert := func(cn string, extKeyUsage x509.ExtKeyUsage) tls.Certificate {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate key for %s: %v", cn, err)
		}
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(time.Now().UnixNano()),
			Subject:      pkix.Name{CommonName: cn, Organization: []string{"VINCULA Test"}},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{extKeyUsage},
			DNSNames:     []string{"localhost"},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		}

		certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		if err != nil {
			t.Fatalf("create cert for %s: %v", cn, err)
		}

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			t.Fatalf("marshal key for %s: %v", cn, err)
		}
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

		tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			t.Fatalf("x509 key pair for %s: %v", cn, err)
		}
		return tlsCert
	}

	return &tlsBundle{
		caPool:     caPool,
		serverCert: issueCert("vincula-server", x509.ExtKeyUsageServerAuth),
		clientCert: issueCert("vincula-client", x509.ExtKeyUsageClientAuth),
	}
}

// ---------------------------------------------------------------------------
// Test-scoped gRPC server helpers
// ---------------------------------------------------------------------------

// testServer wraps a running gRPC server so tests can obtain a connected
// client and shut down cleanly.
type testServer struct {
	addr   string
	server *grpc.Server
}

// startServer creates an in-process gRPC server with mTLS, wires up the
// ClinicalRecordService backed by an in-memory repository, and starts
// serving on a random port.
func startServer(t *testing.T, bundle *tlsBundle) *testServer {
	t.Helper()

	// Server-side TLS with mutual auth.
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{bundle.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    bundle.caPool,
		MinVersion:   tls.VersionTLS13,
	}

	repo := storage.NewInMemoryClinicalRepo()
	uc := usecases.NewClinicalUseCase(repo, nil)
	svc := grpcadapter.NewClinicalServer(uc)

	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	clinicalv1.RegisterClinicalRecordServiceServer(srv, svc)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			// Server was stopped; ignore.
		}
	}()

	t.Cleanup(func() { srv.GracefulStop() })

	return &testServer{addr: lis.Addr().String(), server: srv}
}

// dial creates a gRPC client connection to the test server using the
// provided client certificate.
func dial(t *testing.T, addr string, bundle *tlsBundle) clinicalv1.ClinicalRecordServiceClient {
	t.Helper()

	clientTLS := &tls.Config{
		Certificates: []tls.Certificate{bundle.clientCert},
		RootCAs:      bundle.caPool,
		MinVersion:   tls.VersionTLS13,
		ServerName:   "localhost",
	}

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return clinicalv1.NewClinicalRecordServiceClient(conn)
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// TestRecordAndGetSummary verifies the full write-then-read flow:
//  1. Record several clinical events (allergy, diagnosis, prescription).
//  2. Retrieve the patient summary and assert it contains the expected data.
func TestRecordAndGetSummary(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)
	client := dial(t, ts.addr, bundle)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	patientRun := "12345678-9"

	// --- Record an allergy event ---
	allergyResp, err := client.RecordClinicalEvent(ctx, &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       patientRun,
		EventType:        "allergy",
		EventDataJson:    []byte(`{"alergia":"Penicilina","severidad":"alta"}`),
		AuthorCredential: "DR-INTEGRA-TEST",
		EventTimestamp:   timestamppb.Now(),
	})
	if err != nil {
		t.Fatalf("RecordClinicalEvent (allergy): %v", err)
	}
	if allergyResp.EventId == "" {
		t.Fatal("expected non-empty EventId for allergy event")
	}
	t.Logf("✅ Evento alergia registrado: ID=%s", allergyResp.EventId)

	// --- Record a diagnosis event ---
	diagResp, err := client.RecordClinicalEvent(ctx, &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       patientRun,
		EventType:        "diagnosis",
		EventDataJson:    []byte(`{"diagnostico":"Hipertensión arterial","codigo_cie10":"I10"}`),
		AuthorCredential: "DR-INTEGRA-TEST",
		EventTimestamp:   timestamppb.Now(),
	})
	if err != nil {
		t.Fatalf("RecordClinicalEvent (diagnosis): %v", err)
	}
	if diagResp.EventId == "" {
		t.Fatal("expected non-empty EventId for diagnosis event")
	}
	t.Logf("✅ Evento diagnóstico registrado: ID=%s", diagResp.EventId)

	// --- Record a prescription event ---
	prescResp, err := client.RecordClinicalEvent(ctx, &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       patientRun,
		EventType:        "prescription",
		EventDataJson:    []byte(`{"medicamento":"Losartán 50mg","posologia":"1 comprimido cada 12h"}`),
		AuthorCredential: "DR-INTEGRA-TEST",
		EventTimestamp:   timestamppb.Now(),
	})
	if err != nil {
		t.Fatalf("RecordClinicalEvent (prescription): %v", err)
	}
	if prescResp.EventId == "" {
		t.Fatal("expected non-empty EventId for prescription event")
	}
	t.Logf("✅ Evento prescripción registrado: ID=%s", prescResp.EventId)

	// --- Get patient summary and verify aggregated data ---
	summary, err := client.GetPatientSummary(ctx, &clinicalv1.GetPatientSummaryRequest{
		PatientRun:          patientRun,
		RequesterHospitalId: "HOSP-TEST",
	})
	if err != nil {
		t.Fatalf("GetPatientSummary: %v", err)
	}

	if summary.PatientRun != patientRun {
		t.Errorf("PatientRun = %q, want %q", summary.PatientRun, patientRun)
	}
	if len(summary.ActiveAllergies) == 0 {
		t.Error("expected at least 1 active allergy")
	}
	if len(summary.ActiveDiagnoses) == 0 {
		t.Error("expected at least 1 active diagnosis")
	}
	if len(summary.ActiveMedications) == 0 {
		t.Error("expected at least 1 active medication")
	}
	if summary.LastUpdate == nil {
		t.Error("expected non-nil LastUpdate")
	}
	t.Logf("✅ Resumen del paciente: alergias=%v, diagnósticos=%v, medicamentos=%v",
		summary.ActiveAllergies, summary.ActiveDiagnoses, summary.ActiveMedications)
}

// TestListClinicalEvents verifies that events can be listed with and
// without a type filter.
func TestListClinicalEvents(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)
	client := dial(t, ts.addr, bundle)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	patientRun := "98765432-1"

	// Seed events.
	types := []string{"allergy", "diagnosis", "allergy", "prescription"}
	for i, et := range types {
		_, err := client.RecordClinicalEvent(ctx, &clinicalv1.RecordClinicalEventRequest{
			PatientRun:       patientRun,
			EventType:        et,
			EventDataJson:    []byte(`{"dato":"test"}`),
			AuthorCredential: "DR-LIST-TEST",
			EventTimestamp:   timestamppb.New(time.Now().Add(time.Duration(i) * time.Second)),
		})
		if err != nil {
			t.Fatalf("seed event %d (%s): %v", i, et, err)
		}
	}

	// List all events (no filter).
	resp, err := client.ListClinicalEvents(ctx, &clinicalv1.ListClinicalEventsRequest{
		PatientRun: patientRun,
		PageSize:   100,
	})
	if err != nil {
		t.Fatalf("ListClinicalEvents (all): %v", err)
	}
	if got := len(resp.Events); got != 4 {
		t.Errorf("ListClinicalEvents (all) returned %d events, want 4", got)
	}
	t.Logf("✅ ListClinicalEvents sin filtro devolvió %d eventos", len(resp.Events))

	// List with type filter.
	filtered, err := client.ListClinicalEvents(ctx, &clinicalv1.ListClinicalEventsRequest{
		PatientRun:      patientRun,
		EventTypeFilter: "allergy",
		PageSize:        100,
	})
	if err != nil {
		t.Fatalf("ListClinicalEvents (allergy filter): %v", err)
	}
	if got := len(filtered.Events); got != 2 {
		t.Errorf("ListClinicalEvents (allergy filter) returned %d events, want 2", got)
	}
	t.Logf("✅ ListClinicalEvents con filtro 'allergy' devolvió %d eventos", len(filtered.Events))
}

// TestRevokeConsent verifies the RevokeConsent RPC returns a successful
// response.
func TestRevokeConsent(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)
	client := dial(t, ts.addr, bundle)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RevokeConsent(ctx, &clinicalv1.RevokeConsentRequest{
		PatientRun:   "11111111-1",
		DataCategory: "clinical_history",
	})
	if err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}
	if !resp.Success {
		t.Errorf("RevokeConsent returned success=false, message=%q", resp.Message)
	}
	t.Log("✅ RevokeConsent completado exitosamente")
}

// TestUnauthenticatedClientRejected verifies that a client without a valid
// certificate cannot call the service.
func TestUnauthenticatedClientRejected(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)

	// Connect WITHOUT a client certificate — only trust the CA for the server
	// cert but do not present one ourselves.
	noClientCertTLS := &tls.Config{
		RootCAs:    bundle.caPool,
		MinVersion: tls.VersionTLS13,
		ServerName: "localhost",
	}

	conn, err := grpc.NewClient(
		ts.addr,
		grpc.WithTransportCredentials(credentials.NewTLS(noClientCertTLS)),
	)
	if err != nil {
		t.Fatalf("dial (unauthenticated): %v", err)
	}
	defer conn.Close()

	client := clinicalv1.NewClinicalRecordServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.GetPatientSummary(ctx, &clinicalv1.GetPatientSummaryRequest{
		PatientRun:          "12345678-9",
		RequesterHospitalId: "HOSP-UNAUTH",
	})
	if err == nil {
		t.Fatal("expected error for unauthenticated client, got nil")
	}

	// The server requires mTLS so the TLS handshake itself should fail,
	// surfacing as Unavailable or an equivalent transport error.
	st, ok := status.FromError(err)
	if !ok {
		// Non-gRPC error (e.g. raw transport error) is also acceptable
		// since TLS handshake fails before gRPC framing.
		t.Logf("✅ Conexión sin certificado rechazada (error de transporte): %v", err)
		return
	}

	if st.Code() != codes.Unavailable && st.Code() != codes.Unauthenticated {
		t.Errorf("unexpected gRPC code %v; want Unavailable or Unauthenticated", st.Code())
	}
	t.Logf("✅ Conexión sin certificado rechazada con código: %s", st.Code())
}

// TestRecordEventIdempotency verifies that recording the same event data
// twice produces two distinct event IDs (i.e. the server generates unique
// IDs on each call).
func TestRecordEventIdempotency(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)
	client := dial(t, ts.addr, bundle)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &clinicalv1.RecordClinicalEventRequest{
		PatientRun:       "55555555-5",
		EventType:        "allergy",
		EventDataJson:    []byte(`{"alergia":"Ibuprofeno"}`),
		AuthorCredential: "DR-IDEMPOTENT",
		EventTimestamp:   timestamppb.Now(),
	}

	resp1, err := client.RecordClinicalEvent(ctx, req)
	if err != nil {
		t.Fatalf("first record: %v", err)
	}

	resp2, err := client.RecordClinicalEvent(ctx, req)
	if err != nil {
		t.Fatalf("second record: %v", err)
	}

	if resp1.EventId == resp2.EventId {
		t.Errorf("two records produced the same EventId %q", resp1.EventId)
	}
	t.Logf("✅ IDs únicos generados: %s != %s", resp1.EventId, resp2.EventId)
}

// TestGetSummaryEmptyPatient verifies that requesting a summary for a
// patient with no events returns an empty (but valid) summary.
func TestGetSummaryEmptyPatient(t *testing.T) {
	bundle := newTLSBundle(t)
	ts := startServer(t, bundle)
	client := dial(t, ts.addr, bundle)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	summary, err := client.GetPatientSummary(ctx, &clinicalv1.GetPatientSummaryRequest{
		PatientRun:          "00000000-0",
		RequesterHospitalId: "HOSP-EMPTY",
	})
	if err != nil {
		t.Fatalf("GetPatientSummary (empty): %v", err)
	}

	if summary.PatientRun != "00000000-0" {
		t.Errorf("PatientRun = %q, want %q", summary.PatientRun, "00000000-0")
	}
	if len(summary.ActiveAllergies) != 0 {
		t.Errorf("expected 0 allergies, got %d", len(summary.ActiveAllergies))
	}
	if len(summary.ActiveDiagnoses) != 0 {
		t.Errorf("expected 0 diagnoses, got %d", len(summary.ActiveDiagnoses))
	}
	if len(summary.ActiveMedications) != 0 {
		t.Errorf("expected 0 medications, got %d", len(summary.ActiveMedications))
	}
	t.Log("✅ Resumen vacío retornado correctamente para paciente sin eventos")
}
