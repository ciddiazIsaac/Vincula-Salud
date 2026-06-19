# Tests

## Pruebas de Integración

Las pruebas de integración levantan un servidor gRPC **in-process** con mTLS
(certificados generados en memoria) y un repositorio in-memory. No requieren
infraestructura externa (Spanner, Pub/Sub, certificados en disco).

### Ejecutar

```bash
go test -tags=integration -v ./tests/integration/...
```

### Tests incluidos

| Test | Descripción |
|------|-------------|
| `TestRecordAndGetSummary` | Registra eventos (alergia, diagnóstico, prescripción) y verifica que `GetPatientSummary` agrega los datos correctamente |
| `TestListClinicalEvents` | Verifica listado con y sin filtro por tipo de evento |
| `TestRevokeConsent` | Verifica que la revocación de consentimiento retorna éxito |
| `TestUnauthenticatedClientRejected` | Verifica que un cliente sin certificado es rechazado por mTLS |
| `TestRecordEventIdempotency` | Verifica que cada registro genera un `EventId` único |
| `TestGetSummaryEmptyPatient` | Verifica que un paciente sin eventos retorna un resumen vacío válido |

---

## Pruebas de Carga

### Opción 1: k6 (gRPC nativo)

Requiere [k6](https://k6.io/) instalado y el servidor corriendo localmente con mTLS.

```bash
k6 run \
  --ssl-client-cert certs/client.crt \
  --ssl-client-key certs/client.key \
  tests/load/grpc_load_test.js
```

O usando el target del Makefile:

```bash
make load-test
```

### Opción 2: ghz (gRPC nativo)

Requiere [ghz](https://ghz.sh/) instalado:

```bash
go install github.com/bojand/ghz/cmd/ghz@latest
```

Ejecutar:

```bash
# Valores por defecto: 2000 requests, 50 concurrentes, 100 RPS
bash tests/load/run-load-test.sh

# Personalizar
bash tests/load/run-load-test.sh --rps 200 -n 5000 -c 100
```

### Opción 3: hey (HTTP — solo si hay gRPC-Gateway)

Requiere [hey](https://github.com/rakyll/hey) instalado:

```bash
go install github.com/rakyll/hey@latest
```

Ejecutar:

```bash
bash tests/load/run-hey.sh
bash tests/load/run-hey.sh -n 5000 -c 100 --rps 200
```

> **Nota:** `run-hey.sh` solo es útil si tienes un proxy HTTP/gRPC-Gateway
> configurado. Para testing directo sobre gRPC, usa `ghz` o `k6`.
