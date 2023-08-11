#!/bin/bash

set -eux

export IMAGE=gcr.io/k8s-minikube/gopogh-server

docker buildx build --platform linux/amd64 -t "${IMAGE}" -f Dockerfile.server .

docker push "${IMAGE}" || exit 2

gcloud run deploy gopogh-server \
    --project k8s-minikube \
    --image "${IMAGE}" \
    --set-env-vars="DB_HOST=${DB_HOST},DB_PATH=${DB_PATH}" \
    --allow-unauthenticated \
    --region us-central1 \
    --memory 4Gi \
    --platform managed
