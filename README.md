<div align="center">
  <img src="docs/logo.png" alt="VINCULA Salud Logo" width="200"/>
  <h1>VINCULA Salud</h1>
  <p><b>Plataforma de Interoperabilidad Sanitaria</b></p>

  [![CI](https://github.com/ciddiazIsaac/Vincula-Salud/actions/workflows/ci.yaml/badge.svg)](https://github.com/ciddiazIsaac/Vincula-Salud/actions/workflows/ci.yaml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/ciddiazIsaac/Vincula-Salud)](https://goreportcard.com/report/github.com/ciddiazIsaac/Vincula-Salud)
  [![codecov](https://codecov.io/gh/ciddiazIsaac/Vincula-Salud/branch/main/graph/badge.svg)](https://codecov.io/gh/ciddiazIsaac/Vincula-Salud)
</div>

<br/>

**VINCULA Salud** es un servicio gRPC de alto rendimiento que permite a hospitales del sistema de salud pública de Chile compartir datos clínicos de pacientes de forma segura, estandarizada y en tiempo real. Actúa como el eje central de interoperabilidad: recibe eventos clínicos (diagnósticos, alergias, medicamentos), los persiste en Google Cloud Spanner, y expone resúmenes de historial consolidados a cualquier hospital autorizado de la red.

> **Estado:** Piloto técnico — No usar en producción sin revisión SRE.

---

## 🚀 Inicio Rápido (5 pasos)

Para probar la plataforma localmente utilizando Docker Compose:

1. **Clona el repositorio**
   ```bash
   git clone https://github.com/ciddiazIsaac/Vincula-Salud.git
   cd Vincula-Salud
   ```

2. **Configura el entorno**
   ```bash
   cp .env.example .env
   ```

3. **Genera los certificados mTLS de prueba**
   ```bash
   make gen-certs
   ```
   *(Asegúrate de tener openssl instalado. Esto creará los archivos en la carpeta `certs/`)*

4. **Levanta la infraestructura**
   ```bash
   docker-compose up -d
   ```
   *(Esto iniciará el Servidor gRPC, Emulador de Spanner, Jaeger, Prometheus y Grafana)*

5. **Prueba la API con los clientes de ejemplo**
   Ejecuta el cliente de prueba escrito en Go:
   ```bash
   go run examples/client/main.go
   ```
   *También dispones de ejemplos en [Python](examples/client-python/) y [Node.js](examples/client-node/).*

---

## 📚 Enlaces a Documentación

Hemos movido la documentación detallada a la carpeta `docs/` para facilitar su lectura:

- **[Arquitectura y Stack Tecnológico](docs/architecture.md)**: Detalles sobre el diseño del sistema, diagramas y estructura del proyecto.
- **[Referencia de API (gRPC)](docs/api_reference.md)**: RPCs expuestos y ejemplos de uso.
- **[Guía de Desarrollo](docs/development.md)**: Instrucciones detalladas para desarrolladores (emuladores locales, pruebas, etc.).
- **[Operaciones y Observabilidad](docs/operations.md)**: Métricas, trazas, health checks y alertas.
- **[Despliegue (K8s / Terraform)](docs/deployment.md)**: Cómo desplegar en producción usando GKE y Kubernetes.

### 🤝 Comunidad y Contribución

- Revisa nuestra [Guía de Contribución](CONTRIBUTING.md) para saber cómo reportar bugs o enviar Pull Requests.
- Consulta el [Código de Conducta](CODE_OF_CONDUCT.md).
- El historial de versiones está disponible en el [CHANGELOG](CHANGELOG.md).

> **Licencia**: Proyecto interno del Ministerio de Salud de Chile (MINSAL). Uso restringido. Revisa el archivo [LICENSE](LICENSE) para más detalles.
