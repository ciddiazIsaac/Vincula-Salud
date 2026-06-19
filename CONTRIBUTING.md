# Contribuir a VINCULA Salud

¡Gracias por tu interés en contribuir a VINCULA Salud! Al ser un proyecto interno del Ministerio de Salud de Chile (MINSAL), seguimos procesos estructurados para mantener la calidad, seguridad y estabilidad de la plataforma.

## Reportar bugs
Si encuentras un error o tienes una sugerencia de mejora:
1. Revisa si el problema ya ha sido reportado en los *Issues*.
2. Abre un issue nuevo utilizando la plantilla adecuada.
3. Incluye todos los detalles posibles (logs, pasos para reproducir, entorno).

## Enviar un PR (Pull Request)
Todo el desarrollo se realiza a través de Pull Requests. Para contribuir con código:

1. Crea un fork o trabaja en una rama local (si tienes permisos) con el prefijo adecuado:
   - `feature/nombre-de-la-feature`
   - `fix/descripcion-del-fix`
   - `chore/tarea-de-mantenimiento`
2. Realiza tus cambios asegurándote de seguir los [Estándares de código](#estándares-de-código).
3. Asegúrate de que las pruebas pasen ejecutando `make test`.
4. Si agregas nueva funcionalidad, incluye pruebas unitarias o de integración.
5. Envía el PR apuntando a la rama `main` y solicita la revisión de al menos un mantenedor.

## Estándares de código
- Sigue la guía de estilo estándar de Go.
- Usa `gofmt` y `goimports` antes de hacer commit.
- Documenta las funciones públicas, interfaces y paquetes.
- Mantén la cobertura de pruebas alta (ejecuta `make test` para verificar).
- Asegúrate de que no haya problemas de linters ejecutando `make lint`.
