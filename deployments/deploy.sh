#!/bin/bash
set -e

ENV=${1:-dev}
IMAGE_TAG=${2:-latest}

echo "Deploying VINCULA Salud to $ENV environment"

# Construir imagen Docker
docker build -t gcr.io/vincula-salud-$ENV/clinical-record:$IMAGE_TAG -f Dockerfile .

# Push a Google Container Registry
docker push gcr.io/vincula-salud-$ENV/clinical-record:$IMAGE_TAG

# Aplicar manifests con Kustomize
kubectl apply -k deployments/kubernetes/overlays/$ENV

# Esperar rollout
kubectl rollout status deployment/clinical-record-service -n vinca-$ENV

echo "Deployment complete!"
