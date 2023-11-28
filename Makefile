# Variables
IMAGE_BASE 			:= sodotheil/wish
WISHAPI_VERSION		:= 0.0.1
WISHAPI_IMAGE		:= $(IMAGE_BASE)/wishapi:$(WISHAPI_VERSION)
KIND_IMAGE			:= kindest/node:v1.27.3
KIND_CLUSTER		:= wish-cluster
WISHAPI_APP			:= api-server
NAMESPACE			:= wish

# Local
run:
	go run cmd/wishapi/main.go

tidy:
	go mod tidy

# Containers
wishapi:
	docker build -f infra/docker/Dockerfile -t $(WISHAPI_IMAGE) .

# Kubernetes
dev-up:
	kind create cluster \
		--image $(KIND_IMAGE) \
		--name $(KIND_CLUSTER) \
		--config infra/k8s/dev/kind-config.yaml

dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

dev-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

dev-load:
	kind load docker-image $(WISHAPI_IMAGE) --name $(KIND_CLUSTER)

dev-apply:
	kubectl kustomize infra/k8s/dev/wishapi | kubectl apply -f -

dev-restart:
	kubectl rollout restart deployment $(WISHAPI_APP) --namespace=$(NAMESPACE)

dev-update: wishapi dev-load dev-restart

dev-update-apply: wishapi dev-load dev-apply

dev-logs:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(WISHAPI_APP) --all-containers=true -f --tail=100 --max-log-requests=6 | go run cmd/zapformat/main.go

