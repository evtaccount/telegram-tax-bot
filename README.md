# Telegram Tax Bot

Бот для расчёта налогового резидентства. Запуск в Docker:

## Сборка и запуск

```bash
docker-compose up --build -d
```

## Переменные

- `TELEGRAM_BOT_TOKEN` — токен бота, можно задать через `.env` или Docker secret

## Функции

- Загрузка JSON-файла
- Команды: /start, /help, /reset, /undo
- Добавление / редактирование периодов
- Выгрузка отчёта
- История на диске
