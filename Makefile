.PHONY: help cluster argocd argocd-password argocd-appset argocd-token clean setup-local 

help:
	@echo "Available commands to ease the operations:"
	@echo "- make cluster - to create the kind cluster with the associated configuration"
	@echo "- make argocd - to deploy the official argocd helm chart with the override config"
	@echo "- make argocd-password - to retrieve the argocd admin password"
	@echo "- make argocd-appset - apply the nginx-app ApplicationSet"
	@echo "- make argocd-token - retreive argocd token to use argocd sync"
	@echo "- make setup-local combo command to setup the entire dev infrastructure"
	@echo "- make clean - to delete kind cluster"

cluster:
	kind create cluster --config ./k8s/kind-cluster.yaml

argocd:
	helm repo add argo https://argoproj.github.io/argo-helm
	helm repo update
	helm upgrade --install argocd argo/argo-cd --namespace argocd --create-namespace --version 9.4.17 -f ./k8s/override.argocd.yaml

argocd-password:
	@echo "Waiting for argocd-initial-admin-secret..."
	@until kubectl -n argocd get secret argocd-initial-admin-secret >/dev/null 2>&1; do \
		sleep 3; \
	done
	@kubectl -n argocd get secret argocd-initial-admin-secret \
		-o jsonpath="{.data.password}" | base64 -d && echo

argocd-appset:
	@echo "Waiting for ArgoCD CRDs to be ready..."
	@until kubectl get crd applicationsets.argoproj.io >/dev/null 2>&1; do \
		sleep 3; \
	done
	kubectl apply -f ./k8s/applicationset.yaml

argocd-token:
	@echo "Waiting for argocd-server to be ready..."
	@kubectl rollout status deployment/argocd-server -n argocd --timeout=120s
	@PASS=$$(kubectl -n argocd get secret argocd-initial-admin-secret \
		-o jsonpath="{.data.password}" | base64 -d); \
	curl -sk -X POST https://localhost:8443/api/v1/session \
		-H "Content-Type: application/json" \
		-d "{\"username\":\"admin\",\"password\":\"$$PASS\"}" | jq -r .token

setup-local: cluster argocd argocd-password argocd-appset argocd-token

clean:
	kind delete cluster --name argocd-sync