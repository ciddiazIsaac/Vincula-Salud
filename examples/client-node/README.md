# Cliente Node.js para VINCULA Salud

Este es un ejemplo de cómo consumir la API gRPC de VINCULA Salud utilizando Node.js y mTLS con carga dinámica de Protocol Buffers.

## Prerrequisitos

1. Generar los certificados de prueba. En la raíz del proyecto, ejecuta:
   ```bash
   make gen-certs
   ```
   *(Asegúrate de que los certificados existan en `../../certs/` relativos a este directorio).*

2. Instalar dependencias:
   ```bash
   npm install
   ```

## Ejecutar el cliente

El cliente Node.js carga el archivo `.proto` dinámicamente en tiempo de ejecución, por lo que **no es necesario** compilar el código previamente.

Ejecuta el script (asegúrate de que el servidor gRPC esté corriendo en `localhost:50051`):

```bash
npm start
```
