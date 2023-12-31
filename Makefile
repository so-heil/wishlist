# Variables
IMAGE_BASE 			:= sodotheil/wish
VERSION				:= 0.0.1
WISHAPI_IMAGE		:= $(IMAGE_BASE)/wishapi:$(VERSION)
LIVE_IMAGE			:= $(IMAGE_BASE)/live:$(VERSION)
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
all: admin wishapi live

live:
	docker build -f infra/docker/Dockerfile --target live -t $(LIVE_IMAGE) .

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
	kind load docker-image $(LIVE_IMAGE) --name $(KIND_CLUSTER)

apply-wishapi:
	kubectl kustomize infra/k8s/dev/wishapi | kubectl apply -f -

apply-wishapi-live:
	kubectl kustomize infra/k8s/dev/wishapi-live | kubectl apply -f -

dev-apply-other:
	kubectl apply -f infra/k8s/dev/zipkin/zipkin.yaml
	kubectl apply -f infra/k8s/dev/postgres/postgres.yaml

dev-apply: dev-apply-other apply-wishapi

dev-apply-live: dev-apply-other apply-wishapi-live

dev-restart:
	kubectl rollout restart deployment $(WISHAPI_APP) --namespace=$(NAMESPACE)

dev-hard-restart: dev-down dev-up all dev-load dev-apply-live

dev-update: all dev-load dev-restart

dev-update-apply: all dev-load dev-apply

dev-logs:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(WISHAPI_APP) --all-containers=true -f --tail=100 --max-log-requests=6 | go run cmd/zapformat/main.go

