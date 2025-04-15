# Makefile для управления ботом в Docker

IMAGE_NAME := telegram-tax-bot
SERVICE := tax-bot

build:
	docker-compose build --no-cache

up:
	docker-compose up -d

down:
	docker-compose down

rebuild: down build up

logs:
	docker logs -f $(SERVICE)

reset:
	rm -rf data/* logs/*

cleanup:
	\tdocker stack rm telegram-tax-bot || true
	\tdocker-compose down || true
	\tdocker rm -f $(shell docker ps -aq) || true
	\tdocker rmi telegram-tax-bot telegram-tax-bot-tax-bot || true
	\tdocker image prune -f
	\tdocker network rm telegram-tax-bot_default || true