#!/usr/bin/env bash

#kind create cluster --config kind-config.yaml
#sleep 10

#kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

#kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/refs/heads/main/deploy/static/provider/do/deploy.yaml
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.16.2/cert-manager.yaml

#kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[{"op": "remove", "path": "/spec/template/spec/tolerations"}]'
#kubectl patch deployment ingress-nginx-controller -n ingress-nginx --type=json -p='[{"op": "remove", "path": "/spec/template/spec/nodeSelector"}]'
#
#
#echo "127.0.0.1 echoserver.local" | sudo tee -a /etc/hosts
