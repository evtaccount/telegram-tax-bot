#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────
REMOTE_HOST="206.168.212.122"
REMOTE_USER="evt_tunnel"
REMOTE_PORT="12420"
SSH_KEY="$HOME/.ssh/vpn_key"
REMOTE_DIR="/home/evt_tunnel/telegram-tax-bot"
BRANCH="${1:-master}"

SSH_CMD="ssh -i $SSH_KEY -p $REMOTE_PORT $REMOTE_USER@$REMOTE_HOST"

# ─── Colors ──────────────────────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()   { echo -e "${GREEN}[deploy]${NC} $1"; }
warn()  { echo -e "${YELLOW}[deploy]${NC} $1"; }
error() { echo -e "${RED}[deploy]${NC} $1"; exit 1; }

# ─── Pre-flight checks ──────────────────────────────────────
log "Checking SSH connection..."
$SSH_CMD "echo ok" > /dev/null 2>&1 || error "Cannot connect to server"

log "Deploying branch: $BRANCH"

# ─── Deploy ──────────────────────────────────────────────────
$SSH_CMD << DEPLOY
  set -euo pipefail
  cd "$REMOTE_DIR" || { echo "Directory $REMOTE_DIR not found"; exit 1; }

  echo ">> Pulling latest changes..."
  git fetch origin
  git checkout $BRANCH
  git pull origin $BRANCH

  echo ">> Building and restarting..."
  docker compose up --build -d

  echo ">> Cleaning up old images..."
  docker image prune -f

  echo ">> Status:"
  docker compose ps
DEPLOY

log "Deploy complete!"
