#!/usr/bin/env bash

# when running in KindCluster
if [ -z "$1" ]; then
  kind create cluster --config kind-config.yaml
  sleep 10
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
  kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[{"op": "remove", "path": "/spec/template/spec/tolerations"}]'
  kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[{"op": "remove", "path": "/spec/template/spec/nodeSelector"}]'
else
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/refs/heads/main/deploy/static/provider/do/deploy.yaml
  kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.16.2/cert-manager.yaml
fi
