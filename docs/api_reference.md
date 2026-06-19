# Referencia de API gRPC

> 📚 **Documentación Interactiva (Swagger UI):** Puedes explorar la API a través del [sitio de GitHub Pages configurado](../README.md#enlaces-a-documentación) o abriendo el archivo local `docs/index.html` en tu navegador.

El servicio `ClinicalRecordService` expone los siguientes RPCs, definidos en [`clinical_record.proto`](../api/v1/clinical_record.proto):

| RPC | Descripción | HTTP (gRPC-Gateway) |
|---|---|---|
| `GetPatientSummary` | Obtiene resumen consolidado de un paciente (alergias, diagnósticos, medicamentos) | `GET /v1/patients/{run}/summary` |
| `RecordClinicalEvent` | Registra un evento clínico (diagnóstico, alergia, etc.) | `POST /v1/patients/{run}/events` |
| `ListClinicalEvents` | Lista eventos clínicos con filtro y paginación | `GET /v1/patients/{run}/events` |
| `RevokeConsent` | Revoca consentimiento de un paciente para una categoría de datos | `POST /v1/patients/{run}/consent:revoke` |

## Ejemplo con `grpcurl`

Puedes interactuar con el servidor localmente utilizando `grpcurl` y los certificados generados:

```bash
grpcurl -cacert certs/ca.crt \
  -cert certs/client.crt -key certs/client.key \
  -d '{"patient_run":"12345678-9","event_type":"diagnosis","event_data_json":"eyJkaWFnbm9zdGljbyI6ICJIaXBlcnRlbnNpw7NuIn0=","author_credential":"DR-001"}' \
  localhost:50051 vinca.clinical.v1.ClinicalRecordService/RecordClinicalEvent
```
