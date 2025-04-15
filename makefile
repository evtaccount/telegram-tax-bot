build:
	docker build -t telegram-tax-bot .

run:
	docker-compose up -d

logs:
	docker logs -f tax-bot

stop:
	docker-compose down
