# Variables
IMAGE_BASE 			:= sodotheil/wish
VERSION				:= 0.0.1
WISHAPI_IMAGE		:= $(IMAGE_BASE)/wishapi:$(VERSION)
ADMIN_IMAGE			:= $(IMAGE_BASE)/admin:$(VERSION)
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
all: admin wishapi

wishapi:
	docker build -f infra/docker/Dockerfile --target wishapi -t $(WISHAPI_IMAGE) .

admin:
	docker build -f infra/docker/Dockerfile --target admin -t $(ADMIN_IMAGE) .

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
	kind load docker-image $(ADMIN_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(WISHAPI_IMAGE) --name $(KIND_CLUSTER)

dev-apply:
	kubectl kustomize infra/k8s/dev/wishapi | kubectl apply -f -
	kubectl apply -f infra/k8s/dev/zipkin/zipkin.yaml
	kubectl apply -f infra/k8s/dev/postgres/postgres.yaml

dev-restart:
	kubectl rollout restart deployment $(WISHAPI_APP) --namespace=$(NAMESPACE)

dev-hard-restart: dev-down dev-up all dev-load dev-apply

dev-update: all dev-load dev-restart

dev-update-apply: all dev-load dev-apply

dev-logs:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(WISHAPI_APP) --all-containers=true -f --tail=100 --max-log-requests=6 | go run cmd/zapformat/main.go

