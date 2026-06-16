# VINCULA Salud - Plataforma de Interoperabilidad Sanitaria

**Estado:** Piloto técnico - No usar en producción sin revisión SRE.

## Estructura
- `cmd/` - Puntos de entrada (servidores, workers)
- `internal/` - Código privado (negocio, adapters)
- `pkg/` - Librerías reutilizables (mocks para tests)
- `api/` - Definiciones protobuf/OpenAPI
- `deployments/` - Manifiestos K8s, Terraform

## Configuración inicial
1. Copiar `.env.example` a `.env` y ajustar.
2. Ejecutar `make setup`
3. Instalar gcloud CLI y emuladores (ver docs/development.md)
