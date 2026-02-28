# Changelog

Все значимые изменения в этом проекте будут документированы в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.0.0/),
и этот проект придерживается [Semantic Versioning](https://semver.org/lang/ru/).

## [1.0.1] - 2026-02-14

### Изменения

В список ревьюеров в файле *CODEOWNERS* добавлены:

- @Iamluckyangel
- @Parnishkaspb

## [1.0.0] - 2026-02-06

Первая версия шаблона проекта
для выполнения лабораторных работ по курсу NoSQL баз данных (см. [ndbx](https://github.com/sitnikovik/ndbx))

### Добавлено

- *.env.local*: файл для хранения конфигурации проекта
- *.gitignore*: файл для указания игнорируемых файлов и папок (для популярных языков программирования)
- *.labrc*: файл для указания текущей лабораторной работы
- *CHANGELOG.md*: файл для ведения истории изменений проекта
- *CODEOWNERS*: файл для указания ответственных за различные части проекта
- *CONTRIBUTING.md*: руководство для разработчиков по внесению изменений в проект
- *Makefile*: файл для автоматизации запуска и остановки проекта
- *README.md*: файл с описанием проекта, его целей и инструкциями по использованию
- *.github/workflows/: воркфлоу для автоматической проверки лабораторных работ в GitHub Actions
- *eventhub.yml*: проверка лабораторных работ по проекту EventHub
- *.github/scripts/*: папка для хранения скриптов, используемых в воркфлоу GitHub Actions
- *lab_number.sh*: скрипт для получения номера текущей лабораторной работы из файла *.labrc*

## [2.0.0] - 2026-02-14

### Добавлено

- *cmd/app/main.go*: точка входа приложения
- *config/config.go*: загрузка конфигурации из файла окружения
- *internal/app/app.go*: инициализация и жизненный цикл приложения с graceful shutdown
- *internal/router/handler.go*: HTTP обработчики для API
- *internal/router/ogen/*: автогенерированный код сервера из OpenAPI
- *pkg/httpserver/server.go*: HTTP сервер с graceful shutdown
- *pkg/httpserver/middleware.go*: middleware для CORS, документации и healthcheck
- *pkg/logger/logger.go*: структурированное логирование
- *docs/typespec/*: TypeSpec спецификация для генерации OpenAPI
- *docs/openapi.yaml*: OpenAPI спецификация API
- *docs/redoc.html*: страница документации ReDoc
- *docs/swagger.html*: страница документации Swagger UI
- *Dockerfile*: контейнеризация приложения
- *docker-compose.yml*: оркестрация сервисов
- *.golangci.yml*: конфигурация линтера
- *go.mod*, *go.sum*: зависимости Go проекта

### Изменено

- *README.md*: добавлена документация для проекта
- *.gitignore*: добавлены игнорируемые файлы для Go проектов
- *.env.local*: добавлены переменные PPROF_PORT, LOG_LEVEL
- *Makefile*: добавлены команды для запуска, тестирования и генерации кода

## [2.1.0] - 2026-02-28

### Добавлено

- *internal/router/ogen/oas_json_gen.go*: добавлены JSON-энкодеры/декодеры для новых схем API сессий.
- *internal/router/ogen/oas_interfaces_gen.go*: добавлены интерфейсы операций, сгенерированные из OpenAPI.
- *internal/service/dto/session.go*: добавлены DTO-запросы и DTO-ответы для операций сессий.
- *internal/storage/redis/dto/session.go*: добавлены DTO для хранения сессий в Redis.
- *internal/service/session.go*: добавлена сервисная логика создания и продления сессий.
- *internal/storage/redis/session.go*: добавлена Redis-реализация хранения и продления сессий.
- *pkg/redis/client.go*: добавлен клиент Redis для инициализации подключения и доступа к хранилищу.

### Изменено

- *.env.local*: обновлены локальные переменные окружения для запуска лабораторной работы.
- *.labrc*: обновлён номер/состояние лабораторной работы.
- *config/config.go*: обновлена конфигурация приложения для параметров сессий и Redis.
- *docker-compose.yml*: обновлена конфигурация контейнеров для сервиса и Redis.
- *docs/typespec/main.tsp*: расширена TypeSpec-спецификация методами API сессий.
- *docs/openapi.yaml*: обновлён OpenAPI-контракт под методы `/health` и `/session`.
- *go.mod*: обновлены зависимости, включая переход на `github.com/ovechkin-dm/mockio/v2`.
- *go.sum*: обновлены контрольные суммы зависимостей после изменения модулей.
- *internal/app/app.go*: обновлена инициализация приложения с подключением session-сервиса и Redis-хранилища.
- *internal/router/handler.go*: реализованы методы обработчика API сессий и healthcheck.
- *internal/router/handler_test.go*: добавлены и переработаны HTTP-тесты для `GET /health` и `POST /session`.
- *internal/router/ogen/oas_cfg_gen.go*: обновлена конфигурация сгенерированного Ogen-сервера.
- *internal/router/ogen/oas_client_gen.go*: обновлён клиент Ogen под новый контракт сессий.
- *internal/router/ogen/oas_handlers_gen.go*: обновлена обработка операций Ogen для health/session.
- *internal/router/ogen/oas_operations_gen.go*: обновлён список операций API и их сигнатуры.
- *internal/router/ogen/oas_parameters_gen.go*: обновлены параметры запросов (включая cookie-заголовки).
- *internal/router/ogen/oas_response_decoders_gen.go*: обновлены декодеры ответов для новых операций.
- *internal/router/ogen/oas_response_encoders_gen.go*: обновлены энкодеры ответов для новых операций.
- *internal/router/ogen/oas_router_gen.go*: обновлена маршрутизация Ogen для путей `/health` и `/session`.
- *internal/router/ogen/oas_schemas_gen.go*: обновлены схемы OpenAPI-моделей.
- *internal/router/ogen/oas_server_gen.go*: обновлена серверная обвязка Ogen для новых методов.
- *internal/router/ogen/oas_unimplemented_gen.go*: обновлены заглушки нереализованных методов.

### Удалено

- *internal/router/handler_test.go*: удалён устаревший тест `TestHandler_APIPing` для неактуального эндпоинта `/api/ping`.
