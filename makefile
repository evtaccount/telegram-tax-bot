# ─── Configuration ───────────────────────────────────────────
REMOTE_HOST   := 206.168.212.122
REMOTE_USER   := evt_tunnel
REMOTE_PORT   := 12420
SSH_KEY       := $(HOME)/.ssh/vpn_key
REMOTE_DIR    := /home/evt_tunnel/telegram-tax-bot
BRANCH        ?= master
SERVICE       := tax-bot

SSH_CMD := ssh -i $(SSH_KEY) -p $(REMOTE_PORT) $(REMOTE_USER)@$(REMOTE_HOST)

# ─── Local ───────────────────────────────────────────────────

.PHONY: build test vet run clean

build:
	CGO_ENABLED=0 go build -o bot ./cmd/bot

test:
	go test ./...

vet:
	go vet ./...

run: build
	./bot

clean:
	rm -f bot

# ─── Docker (local) ─────────────────────────────────────────

.PHONY: docker-build docker-up docker-down docker-rebuild docker-logs

docker-build:
	docker compose build --no-cache

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-rebuild: docker-down docker-build docker-up

docker-logs:
	docker logs -f $(SERVICE)

# ─── Deploy ──────────────────────────────────────────────────

.PHONY: deploy deploy-quick ssh status logs-remote

deploy:
	@echo ">> Deploying $(BRANCH) to $(REMOTE_HOST)..."
	$(SSH_CMD) 'cd $(REMOTE_DIR) && git fetch origin && git checkout $(BRANCH) && git pull origin $(BRANCH) && docker compose up --build -d && docker image prune -f && docker compose ps'
	@echo ">> Deploy complete!"

deploy-quick:
	@echo ">> Quick deploy (restart without rebuild)..."
	$(SSH_CMD) 'cd $(REMOTE_DIR) && git pull origin $(BRANCH) && docker compose up -d && docker compose ps'

ssh:
	$(SSH_CMD)

status:
	$(SSH_CMD) 'cd $(REMOTE_DIR) && docker compose ps'

logs-remote:
	$(SSH_CMD) 'cd $(REMOTE_DIR) && docker compose logs --tail=50 $(SERVICE)'

# ─── Cleanup ─────────────────────────────────────────────────

.PHONY: reset cleanup

reset:
	rm -rf data/* logs/*

cleanup:
	docker compose down || true
	docker image prune -f
