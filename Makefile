BASE := ./deployments/docker-compose.yml
DEV := ./deployments/docker-compose.dev.yml
ENV := ./config/.env.dev

BASEF := -f $(BASE) --env-file $(ENV)
DEVF := -f $(BASE) -f $(DEV) --env-file $(ENV)

generate:
	go generate ./...

lint: 
	gofumpt -w cmd/ internal/ pkg/
	golangci-lint run cmd/... internal/... pkg/... 

build: generate
	go build -o gopetbin cmd/gopetbin/main.go

compose-build: generate
	docker compose $(BASEF) build

compose-up:
	docker compose $(BASEF) up -d

compose-down:
	docker compose $(BASEF) down

compose-clean:
	docker compose $(BASEF) run app ./clean


compose-dev-build: generate
	docker compose $(DEVF) build

compose-dev-up: 
	docker compose $(DEVF) up -d

compose-dev-down:
	docker compose $(DEVF) down

compose-dev-clean:
	docker compose $(DEVF) run app ./clean


.PHONY: generate lint build 
