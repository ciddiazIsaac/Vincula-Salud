#!/usr/bin/env bash
# =============================================================================
# VINCULA Salud — HTTP Load Test Runner (hey)
# =============================================================================
#
# Usa "hey" para probar la capa REST/HTTP del gRPC Gateway (si está habilitado).
#
# Prerequisito:
#   go install github.com/rakyll/hey@latest
#
# Uso:
#   ./tests/load/run-hey.sh                    # Parámetros por defecto
#   ./tests/load/run-hey.sh -n 5000 -c 100     # Personalizar
#
# El script asume que:
#   1. El gRPC Gateway (o un proxy HTTP→gRPC) está corriendo en localhost:8443
#      con mTLS habilitado.
#   2. Los certificados mTLS existen en certs/ (ca.crt, client.crt, client.key).
#
# NOTA: Si no tienes un gRPC Gateway habilitado, usa run-load-test.sh con ghz
#       para probar directamente sobre gRPC.
# =============================================================================

set -euo pipefail

# ------------------------------- Defaults ------------------------------------
BASE_URL="${BASE_URL:-https://localhost:8443}"
CERT_DIR="${CERT_DIR:-certs}"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-1000}"
CONCURRENCY="${CONCURRENCY:-20}"
REQUESTS_PER_SECOND="${RPS:-50}"

# Parse overrides.
while [[ $# -gt 0 ]]; do
  case "$1" in
    -n|--total)       TOTAL_REQUESTS="$2"; shift 2 ;;
    -c|--concurrency) CONCURRENCY="$2"; shift 2 ;;
    -q|--rps)         REQUESTS_PER_SECOND="$2"; shift 2 ;;
    --url)            BASE_URL="$2"; shift 2 ;;
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

if ! command -v hey &> /dev/null; then
  echo "❌ 'hey' no está instalado."
  echo "   Instálalo con: go install github.com/rakyll/hey@latest"
  exit 1
fi

echo "============================================================"
echo " VINCULA Salud — Prueba de Carga HTTP (hey)"
echo "============================================================"
echo " Base URL:        $BASE_URL"
echo " Total Requests:  $TOTAL_REQUESTS"
echo " Concurrency:     $CONCURRENCY"
echo " RPS Limit:       $REQUESTS_PER_SECOND"
echo "============================================================"

# ---- Test 1: GET /v1/patients/{run}/summary ----
PATIENT_RUN="LOAD-TEST-001"
ENDPOINT="${BASE_URL}/v1/patients/${PATIENT_RUN}/summary?requesterHospitalId=HOSP-LOAD"

echo ""
echo "📊 Test 1: GET ${ENDPOINT}"
echo "------------------------------------------------------------"
hey -n "${TOTAL_REQUESTS}" \
    -c "${CONCURRENCY}" \
    -q "${REQUESTS_PER_SECOND}" \
    -cert "${CERT_DIR}/client.crt" \
    -key "${CERT_DIR}/client.key" \
    -cacert "${CERT_DIR}/ca.crt" \
    "${ENDPOINT}"

# ---- Test 2: POST /v1/patients/{run}/events ----
ENDPOINT_EVENTS="${BASE_URL}/v1/patients/${PATIENT_RUN}/events"
BODY='{"eventType":"allergy","eventDataJson":"eyJhbGVyZ2lhIjoiUGVuaWNpbGluYSJ9","authorCredential":"DR-LOAD-TEST"}'

echo ""
echo "📊 Test 2: POST ${ENDPOINT_EVENTS}"
echo "------------------------------------------------------------"
hey -n "${TOTAL_REQUESTS}" \
    -c "${CONCURRENCY}" \
    -q "${REQUESTS_PER_SECOND}" \
    -m POST \
    -H "Content-Type: application/json" \
    -d "${BODY}" \
    -cert "${CERT_DIR}/client.crt" \
    -key "${CERT_DIR}/client.key" \
    -cacert "${CERT_DIR}/ca.crt" \
    "${ENDPOINT_EVENTS}"

echo ""
echo "============================================================"
echo " ✅ Prueba de carga HTTP completada"
echo "============================================================"
