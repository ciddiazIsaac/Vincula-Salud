# Política de Seguridad / Security Policy

> Este es un proyecto de salud pública que procesa datos clínicos sensibles.
> Tomamos la seguridad con la máxima seriedad.

---

## Versiones con Soporte / Supported Versions

Solo la rama principal (`main`) recibe parches de seguridad activos.

| Versión / Version | Soporte / Supported |
|-------------------|---------------------|
| `main` (HEAD)     | ✅ Activo            |
| Tags anteriores   | ❌ Sin soporte       |

---

## Reporte Responsable de Vulnerabilidades

**Por favor, NO abra un _issue_ público para reportar vulnerabilidades de seguridad.**
Hacerlo podría exponer a los pacientes y al sistema antes de que el problema sea corregido.

### Cómo Reportar

1. **Envíe un correo electrónico** a la dirección de seguridad del equipo MINSAL responsable
   del proyecto (contacto disponible internamente a través del canal oficial del proyecto).
2. **Incluya en su reporte:**
   - Descripción detallada de la vulnerabilidad.
   - Pasos reproducibles (Proof of Concept, si aplica).
   - Impacto potencial estimado (confidencialidad, integridad, disponibilidad).
   - Versión o commit afectado.
   - Su nombre o alias (para el reconocimiento, si lo desea).

### Qué Esperar

| Plazo                   | Acción                                                         |
|-------------------------|----------------------------------------------------------------|
| **1–2 días hábiles**    | Confirmación de recepción del reporte.                         |
| **7 días hábiles**      | Evaluación inicial de severidad y plan de mitigación.          |
| **30–90 días**          | Corrección, pruebas y despliegue coordinado del parche.        |
| **Post-parche**         | Publicación de aviso de seguridad (si la gravedad lo amerita). |

Trabajaremos con el investigador de forma coordinada y le notificaremos antes de
cualquier divulgación pública.

---

## Alcance / Scope

Las siguientes áreas son prioritarias para reportes de seguridad:

- **Autenticación y autorización** (JWT, mTLS, control de acceso a datos clínicos).
- **Integridad y confidencialidad de datos** (datos del paciente en Cloud Spanner, tránsito gRPC/HTTP).
- **Escalada de privilegios** o acceso no autorizado a eventos clínicos.
- **Inyección** (SQL, gRPC, deserialización insegura de Protobuf).
- **Exposición de secretos** en imágenes Docker, variables de entorno o logs.
- **Dependencias con CVEs conocidos** en `go.mod` o imágenes base.

Las siguientes áreas están **fuera del alcance**:

- Vulnerabilidades en el emulador de Spanner (reportar directamente a Google).
- Ataques de denegación de servicio (DoS) puramente volumétricos sin impacto en datos.
- Problemas de seguridad en entornos de desarrollo/local sin acceso a producción.

---

## Divulgación Coordinada / Coordinated Disclosure

Seguimos el principio de **divulgación responsable coordinada (CVD)**:

1. El equipo recibe el reporte de forma privada.
2. Se desarrolla y prueba el parche de forma interna.
3. Se despliega el parche en producción.
4. Se publica un aviso de seguridad con crédito al investigador (si lo desea).

Solicitamos un período de embargo mínimo de **90 días** desde la notificación
para permitir una remediación adecuada antes de cualquier divulgación pública.

---

## Reconocimiento / Acknowledgements

Agradecemos a todos los investigadores que contribuyan a la seguridad de este proyecto
y al sistema de salud pública de Chile. Los reportes válidos serán reconocidos en el
aviso de seguridad correspondiente.

---

*Este documento está alineado con las mejores prácticas de [coordinated vulnerability disclosure](https://www.cisa.gov/coordinated-vulnerability-disclosure-process) y el estándar [ISO/IEC 29147](https://www.iso.org/standard/72311.html).*
