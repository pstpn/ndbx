# ndbx

Backend-сервис для проекта по NoSQL базам данных.
Проект реализован на Go и включает HTTP API, автоматическую документацию OpenAPI, а также инфраструктуру для профилирования.

## Ключевые возможности

- HTTP API с минимальным набором эндпоинтов для проверки доступности.
- Автогенерация OpenAPI спецификации и клиентского/серверного кода.
- Встроенная документация в Swagger UI и ReDoc.
- Профилирование через pprof.

## Архитектура и структура

- `cmd/app` — точка входа приложения.
- `internal/app` — инициализация сервисов и жизненный цикл.
- `internal/router` — HTTP-обработчики.
- `pkg/httpserver` — инфраструктура HTTP сервера и middleware.
- `config` — загрузка конфигурации из файла окружения.
- `docs` — OpenAPI и статические страницы документации.

## Конфигурация

Приложение читает конфигурацию из файла, путь к которому задается переменной окружения `CONFIG_PATH`.
Формат — пары `KEY=VALUE` (как в `.env`).

Пример `.env.local`:

```env
LOG_LEVEL=info
APP_PORT=8080
PPROF_PORT=6060
```

## Быстрый старт (локально)

```bash
make run-local
```

Команда ожидает, что файл `.env.local` существует в корне проекта.

## Запуск в Docker

```bash
make run
make stop
```

## Документация API

После запуска сервиса доступны следующие страницы:

- ReDoc: `http://localhost:8080/api/docs`
- Swagger UI: `http://localhost:8080/api/swagger`

## Эндпоинты

- Healthcheck: `GET /health`.
- Ping: `GET /api/ping`.

## Профилирование (pprof)

pprof слушает порт `PPROF_PORT` (по умолчанию `6060`). Примеры:

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=10
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/trace?seconds=10
```

## Генерация кода

```bash
make code-gen
```

Команда запускает TypeSpec компиляцию и генерацию серверного кода из OpenAPI.

## Тесты и качество кода

```bash
make test
make lint
```

## Вклад в проект

Правила разработки и формат PR описаны в [CONTRIBUTING.md](CONTRIBUTING.md). Ответственные за ревью — в [CODEOWNERS](CODEOWNERS).
