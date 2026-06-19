#!/usr/bin/env bash
# =============================================================================
# VINCULA Salud — gRPC Load Test Runner (ghz)
# =============================================================================
#
# Prerequisito:
#   go install github.com/bojand/ghz/cmd/ghz@latest
#
# Uso:
#   ./tests/load/run-load-test.sh                    # Parámetros por defecto
#   ./tests/load/run-load-test.sh --rps 200 -n 5000  # Personalizar
#
# El script asume que:
#   1. El servidor gRPC está corriendo en localhost:50051 con mTLS.
#   2. Los certificados mTLS existen en certs/ (ca.crt, client.crt, client.key).
# =============================================================================

set -euo pipefail

# ------------------------------- Defaults ------------------------------------
HOST="${GRPC_HOST:-localhost:50051}"
CERT_DIR="${CERT_DIR:-certs}"
PROTO_PATH="api/v1/clinical_record.proto"
IMPORT_PATHS="-I . -I third_party"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-2000}"
CONCURRENCY="${CONCURRENCY:-50}"
RPS="${RPS:-100}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-summary}"  # summary | csv | json | html | influx-summary

# Parse overrides from CLI flags.
while [[ $# -gt 0 ]]; do
  case "$1" in
    -n|--total)   TOTAL_REQUESTS="$2"; shift 2 ;;
    -c|--concurrency) CONCURRENCY="$2"; shift 2 ;;
    --rps)        RPS="$2"; shift 2 ;;
    --host)       HOST="$2"; shift 2 ;;
    --format)     OUTPUT_FORMAT="$2"; shift 2 ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

# ----------------------------- Validaciones ----------------------------------
for f in "${CERT_DIR}/ca.crt" "${CERT_DIR}/client.crt" "${CERT_DIR}/client.key"; do
  if [[ ! -f "$f" ]]; then
    echo "❌ Certificado no encontrado: $f"
    echo "   Genera los certificados con: make gen-certs"
    exit 1
  fi
done

if ! command -v ghz &> /dev/null; then
  echo "❌ 'ghz' no está instalado."
  echo "   Instálalo con: go install github.com/bojand/ghz/cmd/ghz@latest"
  exit 1
fi

echo "============================================================"
echo " VINCULA Salud — Prueba de Carga gRPC (ghz)"
echo "============================================================"
echo " Host:            $HOST"
echo " Total Requests:  $TOTAL_REQUESTS"
echo " Concurrency:     $CONCURRENCY"
echo " RPS Limit:       $RPS"
echo " Format:          $OUTPUT_FORMAT"
echo "============================================================"

# --------------- Test 1: GetPatientSummary (lectura) -------------------------
echo ""
echo "📊 Test 1: GetPatientSummary (lectura)"
echo "------------------------------------------------------------"
ghz --insecure=false \
    --cacert="${CERT_DIR}/ca.crt" \
    --cert="${CERT_DIR}/client.crt" \
    --key="${CERT_DIR}/client.key" \
    --proto="${PROTO_PATH}" \
    ${IMPORT_PATHS} \
    --call="vinca.clinical.v1.ClinicalRecordService/GetPatientSummary" \
    --data='{"patient_run":"LOAD-TEST-001","requester_hospital_id":"HOSP-LOAD"}' \
    --total="${TOTAL_REQUESTS}" \
    --concurrency="${CONCURRENCY}" \
    --rps="${RPS}" \
    --format="${OUTPUT_FORMAT}" \
    "${HOST}"

# ----------- Test 2: RecordClinicalEvent (escritura) -------------------------
echo ""
echo "📊 Test 2: RecordClinicalEvent (escritura)"
echo "------------------------------------------------------------"
ghz --insecure=false \
    --cacert="${CERT_DIR}/ca.crt" \
    --cert="${CERT_DIR}/client.crt" \
    --key="${CERT_DIR}/client.key" \
    --proto="${PROTO_PATH}" \
    ${IMPORT_PATHS} \
    --call="vinca.clinical.v1.ClinicalRecordService/RecordClinicalEvent" \
    --data='{"patient_run":"LOAD-TEST-001","event_type":"allergy","event_data_json":"eyJhbGVyZ2lhIjoiUGVuaWNpbGluYSJ9","author_credential":"DR-LOAD-TEST"}' \
    --total="${TOTAL_REQUESTS}" \
    --concurrency="${CONCURRENCY}" \
    --rps="${RPS}" \
    --format="${OUTPUT_FORMAT}" \
    "${HOST}"

# ----------- Test 3: ListClinicalEvents (lectura paginada) -------------------
echo ""
echo "📊 Test 3: ListClinicalEvents (lectura paginada)"
echo "------------------------------------------------------------"
ghz --insecure=false \
    --cacert="${CERT_DIR}/ca.crt" \
    --cert="${CERT_DIR}/client.crt" \
    --key="${CERT_DIR}/client.key" \
    --proto="${PROTO_PATH}" \
    ${IMPORT_PATHS} \
    --call="vinca.clinical.v1.ClinicalRecordService/ListClinicalEvents" \
    --data='{"patient_run":"LOAD-TEST-001","page_size":50}' \
    --total="${TOTAL_REQUESTS}" \
    --concurrency="${CONCURRENCY}" \
    --rps="${RPS}" \
    --format="${OUTPUT_FORMAT}" \
    "${HOST}"

echo ""
echo "============================================================"
echo " ✅ Prueba de carga completada"
echo "============================================================"
