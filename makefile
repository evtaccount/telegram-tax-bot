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
	docker stack rm telegram-tax-bot || true
	docker-compose down || true
	docker rm -f $(shell docker ps -aq) || true
	docker rmi telegram-tax-bot telegram-tax-bot-tax-bot || true
	docker image prune -f
	docker network rm telegram-tax-bot_default || true
