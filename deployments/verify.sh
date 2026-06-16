#!/bin/bash

NS=${1:-vinca-dev}

echo "== Verificando despliegue en namespace: $NS =="

# Verificar pods
PODS=$(kubectl get pods -n $NS -l app=vinca,component=clinical-record -o jsonpath='{.items[*].status.phase}')
if [[ "$PODS" != *"Running"* ]]; then
    echo "❌ Pods no están Running"
    kubectl get pods -n $NS -l app=vinca,component=clinical-record
    exit 1
fi
echo "✅ Pods están Running"

# Verificar health check
POD_NAME=$(kubectl get pods -n $NS -l app=vinca,component=clinical-record -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n $NS $POD_NAME -- grpc_health_probe -addr=localhost:50051
if [ $? -eq 0 ]; then
    echo "✅ Health check gRPC OK"
else
    echo "❌ Health check gRPC FAILED"
    exit 1
fi

# Verificar servicio
kubectl get svc -n $NS clinical-record-service
if [ $? -eq 0 ]; then
    echo "✅ Servicio expuesto correctamente"
else
    echo "❌ Servicio no encontrado"
    exit 1
fi

echo "✅ Verificación completada exitosamente"
