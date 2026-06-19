# Cliente Python para VINCULA Salud

Este es un ejemplo de cómo consumir la API gRPC de VINCULA Salud utilizando Python y mTLS.

## Prerrequisitos

1. Generar los certificados de prueba. En la raíz del proyecto, ejecuta:
   ```bash
   make gen-certs
   ```
   *(Asegúrate de que los certificados existan en `../../certs/` relativos a este directorio).*

2. Instalar dependencias:
   ```bash
   pip install -r requirements.txt
   ```

## Generar código gRPC

Antes de ejecutar el script, debes generar el código Python a partir del archivo `.proto`:

```bash
python -m grpc_tools.protoc -I../../ -I../../third_party --python_out=. --grpc_python_out=. ../../api/v1/clinical_record.proto
```

## Ejecutar el cliente

Ejecuta el script (asegúrate de que el servidor gRPC esté corriendo en `localhost:50051`):

```bash
python main.py
```
