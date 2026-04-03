# ndbx

Backend-сервис для проекта по NoSQL базам данных. Проект реализован на Go, использует Redis для сессий, MongoDB для пользователей и событий, а HTTP-контракт и серверная обвязка генерируются из TypeSpec/OpenAPI.

## Возможности

- healthcheck и управление анонимной сессией;
- регистрация пользователей и аутентификация по логину и паролю;
- создание и частичное редактирование событий от имени авторизованного пользователя;
- просмотр событий с фильтрацией по названию, категории, цене, городу, датам и пользователю;
- просмотр профиля пользователя, списка пользователей и событий конкретного пользователя;
- автогенерация OpenAPI и ogen-сервера;
- Swagger UI, ReDoc и pprof.

## Архитектура

- `cmd/app` — точка входа приложения;
- `config` — загрузка конфигурации;
- `internal/app` — инициализация зависимостей и жизненный цикл;
- `internal/router` — HTTP-хендлеры и сгенерированный ogen-код;
- `internal/service` — бизнес-логика;
- `internal/storage/mongodb` — MongoDB-хранилище пользователей и событий;
- `internal/storage/redis` — Redis-хранилище сессий;
- `pkg/httpserver` — HTTP-инфраструктура и общие валидаторы;
- `pkg/logger` — логирование;
- `docs` — TypeSpec, OpenAPI и страницы документации.

## Конфигурация

Приложение читает переменные окружения из файла, путь к которому задаётся через `CONFIG_PATH`. Формат файла — `KEY=VALUE`.

Пример `.env.local`:

```env
LOG_LEVEL=info
APP_HOST=0.0.0.0
APP_PORT=8080
APP_USER_SESSION_TTL=60
PPROF_PORT=6060

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

MONGODB_DATABASE=eventhub
MONGODB_USER=your_mongodb_username
MONGODB_PASSWORD=your_mongodb_password
MONGODB_HOST=mongos
MONGODB_PORT=27017
```

При запуске через Docker Compose MongoDB поднимается как sharded cluster с отдельным `mongos`-роутером.

## Запуск

Локально:

```bash
make run-local
```

В Docker:

```bash
make run
make stop
```

## Документация API

После запуска сервиса доступны:

- ReDoc: `http://localhost:8080/api/docs`
- Swagger UI: `http://localhost:8080/api/swagger`

## Основные endpoint-ы

- `GET /health` — healthcheck;
- `POST /session` — создание или продление анонимной сессии;
- `POST /users` — регистрация пользователя;
- `POST /auth/login` — аутентификация пользователя и привязка сессии;
- `POST /auth/logout` — завершение пользовательской сессии;
- `POST /events` — создание события авторизованным пользователем;
- `GET /events` — получение событий с параметрами `id`, `title`, `category`, `address`, `city`, `price_from`, `price_to`, `date_from`, `date_to`, `user_id`, `user`, `limit`, `offset`;
- `GET /events/{id}` — получение события по идентификатору;
- `PATCH /events/{id}` — частичное обновление события;
- `GET /users` — получение списка пользователей с фильтрацией по `id` и `name`;
- `GET /users/{id}` — получение профиля пользователя;
- `GET /users/{id}/events` — получение событий пользователя с фильтрами по событиям.

## Генерация кода

```bash
make code-gen
```

Команда пересобирает OpenAPI-спецификацию из TypeSpec и обновляет ogen-код сервера.

## Проверки

```bash
make test
make lint
```

## Профилирование

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=10
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/trace?seconds=10
```

## Разработка

Правила внесения изменений описаны в [CONTRIBUTING.md](CONTRIBUTING.md). Ответственные за ревью указаны в [CODEOWNERS](CODEOWNERS).
