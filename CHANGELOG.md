# Changelog

Todos los cambios notables de este proyecto se documentarán en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/),
y este proyecto se adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Ejemplos de clientes gRPC**: Nuevos ejemplos completos en Python (`examples/client-python`) y Node.js (`examples/client-node`) demostrando la conexión segura mediante mTLS.
- **Testing (Fuzz & Mutation)**: Añadida prueba de fuzzing `FuzzValidRunRegex` en el middleware de validación y comando `make mutation-test` en el `Makefile` para evaluar la robustez de los tests.
- **Testing (Load & Integration)**: Nuevos scripts de carga basados en `ghz` y `hey`. Refactorización de las pruebas de integración para ser autocontenidas y levantar el servidor `in-process`.
- **Badges de calidad**: Añadidos badges de CI, Codecov y Go Report Card al `README.md`.
- **Documentación de comunidad**: Añadidos archivos `CONTRIBUTING.md` y `CODE_OF_CONDUCT.md`.
- **Licencia**: Añadido archivo `LICENSE` especificando el uso interno del Ministerio de Salud.

### Changed
- **Configuración local**: Actualizado `.env.example` con todas las variables necesarias (Jaeger, PubSub, Spanner, etc.).
- **CI/CD**: El workflow de GitHub Actions (`ci.yaml`) ahora ejecuta automáticamente las pruebas de integración `make test-integration` además de las unitarias.
- **Docker Compose**: Renombrado `docker-compose.yaml` a `docker-compose.yml` para alinear con las convenciones estándar.
