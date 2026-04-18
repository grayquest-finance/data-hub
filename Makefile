# Data Hub - Go GraphQL Service Makefile
# Usage: make <target>

# ── Variables ──────────────────────────────────────────────────────────────────
APP_NAME        = data-hub
IMAGE_NAME      = $(APP_NAME)
CONTAINER_NAME  = $(APP_NAME)-container
PORT            = 8080
BINARY          = $(APP_NAME)
CMD_PATH        = ./cmd/main.go
GO              = go

# AWS & ECR Configuration
AWS_REGION      = ap-south-1
AWS_ACCOUNT_ID  = 579897422692

# Environment-specific Configuration (can be overridden)
ENVIRONMENT          = uat
ECR_REPOSITORY_NAME  = $(APP_NAME)-$(ENVIRONMENT)
ECR_REPOSITORY_URI   = $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(ECR_REPOSITORY_NAME)

# EKS Configuration
EKS_CLUSTER_NAME   = $(shell \
	if [ "$(ENVIRONMENT)" = "uat" ] || [ "$(ENVIRONMENT)" = "staging" ]; then \
		echo "sandbox-tools"; \
	elif [ "$(ENVIRONMENT)" = "preprod" ]; then \
		echo "pre-prod-cluster"; \
	elif [ "$(ENVIRONMENT)" = "prod" ]; then \
		echo "production-cluster"; \
	else \
		echo "sandbox-tools"; \
	fi)
K8S_NAMESPACE      = $(APP_NAME)-$(ENVIRONMENT)
K8S_DEPLOYMENT     = $(APP_NAME)

# Build Configuration
IMAGE_TAG    ?= latest
MAX_RELEASES ?= 20

# Colors
RED    = \033[0;31m
GREEN  = \033[0;32m
YELLOW = \033[0;33m
BLUE   = \033[0;34m
NC     = \033[0m

.DEFAULT_GOAL := help
.PHONY: help build build-clean run dev stop logs shell test lint vet generate \
        up down clean clean-all status health info \
        ecr-login ecr-build ecr-push ecr-list ecr-cleanup \
        eks-context eks-deploy eks-rollback eks-status eks-wait eks-logs \
        deploy-pipeline deploy-info aws-check

# ── Help ───────────────────────────────────────────────────────────────────────
help:
	@echo "${BLUE}Data Hub - Available Commands:${NC}"
	@echo ""
	@echo "${YELLOW}Local Development:${NC}"
	@echo "  ${GREEN}run${NC}             Run locally (reads .env)"
	@echo "  ${GREEN}dev${NC}             Run with live reload (requires air)"
	@echo "  ${GREEN}generate${NC}        Regenerate gqlgen bindings"
	@echo "  ${GREEN}build${NC}           Build binary"
	@echo ""
	@echo "${YELLOW}Docker Commands:${NC}"
	@echo "  ${GREEN}docker-build${NC}    Build Docker image"
	@echo "  ${GREEN}docker-run${NC}      Run container"
	@echo "  ${GREEN}up${NC}              Start with docker-compose"
	@echo "  ${GREEN}up-cache${NC}        Start with docker-compose + Redis"
	@echo "  ${GREEN}down${NC}            Stop docker-compose services"
	@echo "  ${GREEN}stop${NC}            Stop and remove containers"
	@echo ""
	@echo "${YELLOW}Testing & Quality:${NC}"
	@echo "  ${GREEN}test${NC}            Run all tests"
	@echo "  ${GREEN}test-verbose${NC}    Run tests with verbose output"
	@echo "  ${GREEN}vet${NC}             Run go vet"
	@echo "  ${GREEN}lint${NC}            Run golangci-lint"
	@echo ""
	@echo "${YELLOW}Monitoring:${NC}"
	@echo "  ${GREEN}health${NC}          Check application health"
	@echo "  ${GREEN}logs${NC}            Tail container logs"
	@echo "  ${GREEN}status${NC}          Show container/pod status"
	@echo ""
	@echo "${YELLOW}AWS Deployment (EKS):${NC}"
	@echo "  ${GREEN}deploy-pipeline${NC} Complete build → push → deploy pipeline"
	@echo "  ${GREEN}eks-deploy${NC}      Deploy image to EKS"
	@echo "  ${GREEN}eks-rollback${NC}    Rollback to previous deployment"
	@echo "  ${GREEN}eks-status${NC}      Check EKS deployment status"
	@echo "  ${GREEN}eks-logs${NC}        Tail pod logs from EKS"
	@echo "  ${GREEN}deploy-info${NC}     Show deployment configuration"
	@echo "  ${GREEN}ecr-login${NC}       Login to AWS ECR"
	@echo "  ${GREEN}ecr-list${NC}        List ECR images"

# ── Local Development ──────────────────────────────────────────────────────────
generate: ## Regenerate gqlgen bindings (run after schema changes)
	@echo "${YELLOW}Generating gqlgen bindings...${NC}"
	$(GO) run github.com/99designs/gqlgen generate
	@echo "${GREEN}Generation complete!${NC}"

build: ## Build the binary
	@echo "${YELLOW}Building binary...${NC}"
	$(GO) build -ldflags="-s -w" -o $(BINARY) $(CMD_PATH)
	@echo "${GREEN}Binary built: ./$(BINARY)${NC}"

run: ## Run locally (reads .env automatically via viper)
	@echo "${YELLOW}Starting $(APP_NAME) locally...${NC}"
	$(GO) run $(CMD_PATH)

dev: ## Run with live reload (requires: go install github.com/air-verse/air@latest)
	@echo "${YELLOW}Starting development server with live reload...${NC}"
	@which air > /dev/null 2>&1 || (echo "${RED}air not found. Install: go install github.com/air-verse/air@latest${NC}" && exit 1)
	air

# ── Testing & Quality ──────────────────────────────────────────────────────────
test: ## Run all tests
	@echo "${YELLOW}Running tests...${NC}"
	$(GO) test ./... -count=1
	@echo "${GREEN}Tests completed!${NC}"

test-verbose: ## Run tests with verbose output
	$(GO) test ./... -v -count=1

test-single: ## Run a single test: make test-single TEST=TestFoo PKG=./tests/
	$(GO) test $(PKG) -run $(TEST) -v

vet: ## Run go vet
	@echo "${YELLOW}Running go vet...${NC}"
	$(GO) vet ./...
	@echo "${GREEN}Vet passed!${NC}"

lint: ## Run golangci-lint (requires: brew install golangci-lint)
	@which golangci-lint > /dev/null 2>&1 || (echo "${RED}Install: brew install golangci-lint${NC}" && exit 1)
	golangci-lint run ./...

# ── Docker ─────────────────────────────────────────────────────────────────────
docker-build: ## Build Docker image
	@echo "${YELLOW}Building Docker image...${NC}"
	docker build -t $(IMAGE_NAME):latest .
	@echo "${GREEN}Build complete!${NC}"

build-clean: ## Build Docker image without cache
	docker build --no-cache -t $(IMAGE_NAME):latest .

docker-run: ## Run container
	docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):8080 \
		--env-file .env \
		$(IMAGE_NAME):latest
	@echo "${GREEN}Running on http://localhost:$(PORT)${NC}"

stop: ## Stop and remove containers
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true

logs: ## Tail container logs
	docker logs -f $(CONTAINER_NAME)

shell: ## Open shell in running container
	docker exec -it $(CONTAINER_NAME) /bin/sh

up: ## Start with docker-compose (API only)
	docker-compose up -d

up-cache: ## Start with docker-compose + Redis
	docker-compose --profile cache up -d

down: ## Stop docker-compose services
	docker-compose down

# ── Health & Status ────────────────────────────────────────────────────────────
health: ## Check application health
	@curl -sf http://localhost:$(PORT)/health && echo "${GREEN}Healthy!${NC}" || echo "${RED}Health check failed${NC}"

status: ## Show container and pod status
	@echo "${BLUE}Local Containers:${NC}"
	@docker ps -a | grep $(APP_NAME) || echo "No local containers"
	@echo ""
	@echo "${BLUE}EKS Pods ($(K8S_NAMESPACE)):${NC}"
	@kubectl get pods -n $(K8S_NAMESPACE) -l app=$(APP_NAME) 2>/dev/null || echo "kubectl not configured or namespace not found"

info: ## Show environment information
	@echo "${BLUE}Environment:${NC}"
	@echo "  App:         $(APP_NAME)"
	@echo "  Environment: $(ENVIRONMENT)"
	@echo "  Image Tag:   $(IMAGE_TAG)"
	@echo "  EKS Cluster: $(EKS_CLUSTER_NAME)"
	@echo "  Namespace:   $(K8S_NAMESPACE)"
	@echo "  Go:          $$($(GO) version)"
	@echo "  Docker:      $$(docker --version)"

# ── Cleanup ────────────────────────────────────────────────────────────────────
clean: ## Remove binary, containers, and images
	-rm -f $(BINARY)
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	docker system prune -f

clean-all: ## Remove everything (containers, images, volumes)
	@echo "${RED}WARNING: This will remove all containers, images, and volumes!${NC}"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker system prune -a --volumes -f; \
	else \
		echo "${YELLOW}Cancelled${NC}"; \
	fi

# ── AWS ECR ────────────────────────────────────────────────────────────────────
aws-check: ## Check AWS CLI configuration
	@aws --version
	@aws configure list

ecr-login: aws-check ## Login to AWS ECR
	@echo "${YELLOW}Logging into ECR...${NC}"
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(ECR_REPOSITORY_URI)
	@echo "${GREEN}ECR login successful${NC}"

ecr-build: ## Build and tag image for ECR
	@echo "${YELLOW}Building image for ECR: $(IMAGE_TAG)${NC}"
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(ECR_REPOSITORY_URI):$(IMAGE_TAG)
	@echo "${GREEN}Tagged: $(ECR_REPOSITORY_URI):$(IMAGE_TAG)${NC}"

ecr-push: ## Push image to ECR
	@echo "${YELLOW}Pushing to ECR: $(ECR_REPOSITORY_URI):$(IMAGE_TAG)${NC}"
	docker tag $(IMAGE_NAME):latest $(ECR_REPOSITORY_URI):$(IMAGE_TAG)
	docker push $(ECR_REPOSITORY_URI):$(IMAGE_TAG)
	@echo "${GREEN}Image pushed${NC}"

ecr-list: aws-check ## List ECR images
	aws ecr describe-images --repository-name $(ECR_REPOSITORY_NAME) --region $(AWS_REGION) \
		--query 'imageDetails[*].[imageTags[0],imagePushedAt]' --output table

ecr-cleanup: aws-check ## Clean up old ECR images (keep MAX_RELEASES)
	@echo "${YELLOW}Cleaning up old ECR images (keeping $(MAX_RELEASES))...${NC}"
	@aws ecr describe-images --repository-name $(ECR_REPOSITORY_NAME) --region $(AWS_REGION) \
		--query "sort_by(imageDetails,&imagePushedAt)[:-$(MAX_RELEASES)].imageDigest" \
		--output text | tr '\t' '\n' | grep -v '^$$' > /tmp/images_to_delete.txt || true
	@if [ -s /tmp/images_to_delete.txt ]; then \
		while IFS= read -r digest; do \
			aws ecr batch-delete-image --repository-name $(ECR_REPOSITORY_NAME) --region $(AWS_REGION) \
				--image-ids imageDigest=$$digest || true; \
		done < /tmp/images_to_delete.txt; \
		rm -f /tmp/images_to_delete.txt; \
		echo "${GREEN}Old images cleaned up${NC}"; \
	else \
		echo "${GREEN}No old images to clean up${NC}"; \
	fi

# ── AWS EKS ────────────────────────────────────────────────────────────────────
eks-context: aws-check ## Set kubectl context to EKS cluster
	@echo "${YELLOW}Setting kubectl context to $(EKS_CLUSTER_NAME)...${NC}"
	aws eks update-kubeconfig --region $(AWS_REGION) --name $(EKS_CLUSTER_NAME)
	@echo "${GREEN}kubectl context set${NC}"

eks-deploy: eks-context ## Deploy new image to EKS
	@echo "${YELLOW}Deploying $(ECR_REPOSITORY_URI):$(IMAGE_TAG) to EKS...${NC}"
	kubectl set image deployment/$(K8S_DEPLOYMENT) \
		$(APP_NAME)=$(ECR_REPOSITORY_URI):$(IMAGE_TAG) \
		-n $(K8S_NAMESPACE)
	@echo "${GREEN}Deployment initiated${NC}"

eks-rollback: eks-context ## Rollback EKS deployment to previous version
	@echo "${YELLOW}Rolling back deployment in $(K8S_NAMESPACE)...${NC}"
	kubectl rollout undo deployment/$(K8S_DEPLOYMENT) -n $(K8S_NAMESPACE)
	@echo "${GREEN}Rollback initiated${NC}"

eks-status: eks-context ## Check EKS deployment status
	@echo "${BLUE}Deployment status ($(K8S_NAMESPACE)):${NC}"
	kubectl rollout status deployment/$(K8S_DEPLOYMENT) -n $(K8S_NAMESPACE)
	@echo ""
	kubectl get pods -n $(K8S_NAMESPACE) -l app=$(APP_NAME)

eks-wait: eks-context ## Wait for EKS rollout to complete
	@echo "${YELLOW}Waiting for rollout to complete...${NC}"
	kubectl rollout status deployment/$(K8S_DEPLOYMENT) -n $(K8S_NAMESPACE) --timeout=10m
	@echo "${GREEN}Rollout complete${NC}"

eks-logs: eks-context ## Tail pod logs from EKS
	kubectl logs -f -l app=$(APP_NAME) -n $(K8S_NAMESPACE) --max-log-requests=5

# ── Full Pipeline ──────────────────────────────────────────────────────────────
deploy-pipeline: ecr-build ecr-push eks-deploy ## Full build → push → deploy pipeline
	@echo "${GREEN}Deployment pipeline complete!${NC}"
	@echo "  Image:     $(ECR_REPOSITORY_URI):$(IMAGE_TAG)"
	@echo "  Cluster:   $(EKS_CLUSTER_NAME)"
	@echo "  Namespace: $(K8S_NAMESPACE)"
	@echo "${YELLOW}Run 'make eks-wait' to wait for rollout${NC}"

deploy-info: ## Show deployment configuration
	@echo "${BLUE}=== Data Hub Deployment Configuration ===${NC}"
	@echo "  App:         $(APP_NAME)"
	@echo "  Environment: $(ENVIRONMENT)"
	@echo "  Image Tag:   $(IMAGE_TAG)"
	@echo ""
	@echo "${YELLOW}AWS:${NC}"
	@echo "  Region:      $(AWS_REGION)"
	@echo "  Account:     $(AWS_ACCOUNT_ID)"
	@echo ""
	@echo "${YELLOW}ECR:${NC}"
	@echo "  Repository:  $(ECR_REPOSITORY_NAME)"
	@echo "  Full URI:    $(ECR_REPOSITORY_URI):$(IMAGE_TAG)"
	@echo ""
	@echo "${YELLOW}EKS:${NC}"
	@echo "  Cluster:     $(EKS_CLUSTER_NAME)"
	@echo "  Namespace:   $(K8S_NAMESPACE)"
	@echo "  Deployment:  $(K8S_DEPLOYMENT)"