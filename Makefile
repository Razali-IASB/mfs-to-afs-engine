.PHONY: help build start stop restart logs clean seed test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build all Docker images
	docker-compose build

start: ## Start all services
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services started successfully!"
	@echo "AFS Engine: http://localhost:3000/health"
	@echo "MongoDB UI: http://localhost:8081 (admin/admin)"
	@echo "API Receiver: http://localhost:3001/health"

stop: ## Stop all services
	docker-compose down

restart: ## Restart all services
	docker-compose restart

logs: ## View logs from all services
	docker-compose logs -f

logs-afs: ## View AFS Engine logs only
	docker-compose logs -f afs-engine

logs-mongo: ## View MongoDB logs only
	docker-compose logs -f mongodb-primary

clean: ## Stop and remove all containers, volumes
	docker-compose down -v
	rm -rf logs/* archive/*

seed: ## Seed sample MFS data
	@echo "Seeding sample data..."
	cd scripts && go run seed.go

status: ## Check service status
	@echo "=== Service Status ==="
	@docker-compose ps
	@echo ""
	@echo "=== Health Checks ==="
	@curl -s http://localhost:3000/health | jq . || echo "AFS Engine not responding"
	@echo ""
	@curl -s http://localhost:3001/health | jq . || echo "API Receiver not responding"

stats: ## Show AFS statistics
	@echo "=== AFS Statistics ==="
	@curl -s http://localhost:3000/api/stats | jq .

generate: ## Manually trigger AFS generation for today
	@echo "Triggering manual AFS generation..."
	@curl -X POST http://localhost:3000/api/generate \
		-H "Content-Type: application/json" \
		-d "{\"date\":\"$$(date +%Y-%m-%d)\"}" | jq .

retry: ## Retry failed deliveries
	@echo "Retrying failed deliveries..."
	@curl -X POST http://localhost:3000/api/retry | jq .

mongo-shell: ## Open MongoDB shell
	docker exec -it afs-mongodb-primary mongosh \
		-u admin -p afs_secure_pass_2026 \
		--authenticationDatabase admin afs_db

backup-mongo: ## Backup MongoDB
	@echo "Creating MongoDB backup..."
	@mkdir -p backups
	docker exec afs-mongodb-primary mongodump \
		-u admin -p afs_secure_pass_2026 \
		--authenticationDatabase admin \
		--out=/tmp/backup
	docker cp afs-mongodb-primary:/tmp/backup ./backups/backup-$$(date +%Y%m%d-%H%M%S)
	@echo "Backup completed: ./backups/backup-$$(date +%Y%m%d-%H%M%S)"

test: ## Run tests
	go test ./... -v

dev: ## Run in development mode (local, not Docker)
	go run cmd/afs-engine/main.go
