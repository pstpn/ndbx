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

## [3.0.0] - 2026-03-17

### Добавлено

- *internal/service/dto/event.go*: добавлены DTO для операций создания и чтения событий.
- *internal/service/dto/user.go*: добавлены DTO для регистрации и аутентификации пользователей.
- *internal/storage/mongodb/dto/event.go*: добавлены DTO документов событий MongoDB.
- *internal/storage/mongodb/dto/user.go*: добавлены DTO документов пользователей MongoDB.
- *internal/storage/mongodb/event.go*: добавлено хранилище событий с индексами и фильтрацией.
- *internal/storage/mongodb/user.go*: добавлено хранилище пользователей с уникальным индексом по `username`.
- *pkg/httpserver/validator.go*: добавлены общие валидаторы для обязательных строк, параметров пагинации и RFC3339-дат.

### Изменено

- *README.md*: документация обновлена под лабораторную работу с пользователями, авторизацией и событиями.
- *.env.local*: расширен набор переменных окружения для подключения MongoDB.
- *docker-compose.yml*: обновлена инфраструктура запуска c MongoDB и Redis.
- *docs/typespec/main.tsp*: TypeSpec-спецификация расширена endpoint-ами `/users`, `/auth/login`, `/auth/logout` и `/events`.
- *docs/openapi.yaml*: OpenAPI-контракт обновлён под новые методы пользователей и событий.
- *internal/app/app.go*: инициализация приложения расширена сервисами и storage для MongoDB.
- *internal/router/errors.go*: обновлены ошибки API для пользователей и событий.
- *internal/router/handler.go*: реализованы обработчики регистрации, логина, логаута, создания и получения событий.
- *internal/router/handler_test.go*: добавлены и обновлены HTTP-тесты для пользовательских сценариев и событий.
- *internal/router/ogen/*: ogen-код синхронизирован с новым контрактом OpenAPI.
- *internal/service/event.go*: сервис событий переведён на DTO и доменные ошибки `already exists`.
- *internal/service/session.go*: сервис сессий обновлён для хранения `user_id` и корректной обработки `not found`.
- *internal/service/user.go*: добавлены регистрация пользователя с bcrypt и аутентификация по логину/паролю.
- *internal/storage/redis/session.go*: Redis-хранилище сессий обновлено под поле `user_id`.

## [4.0.0] - 2026-04-03

### Добавлено

- *internal/service/dto/event.go*: расширены DTO событий для фильтров по категории, цене, городу, датам и пользователю.
- *internal/service/dto/user.go*: добавлены DTO для поиска пользователей и просмотра событий пользователя.
- *internal/storage/mongodb/dto/event.go*: расширены DTO документов событий для фильтрации и частичного обновления.
- *internal/storage/mongodb/event.go*: добавлены получение события по id, частичное обновление и расширенная фильтрация событий.
- *internal/storage/mongodb/user.go*: добавлены поиск пользователей и получение пользователя по id.
- *internal/storage/mongodb/mongodb.go*: вынесены общие MongoDB-хелперы для работы с идентификаторами.
- *pkg/httpserver/validator.go*: добавлены общие валидаторы полей для числовых параметров и дат.
- *docker-compose.yml*: добавлена инициализация sharded MongoDB-кластера через отдельный одноразовый сервис.

### Изменено

- *README.md*: документация обновлена под лабораторную работу с пользователями, событиями и фильтрацией.
- *docs/typespec/main.tsp*: TypeSpec-спецификация расширена методами `/users`, `/events/{id}` и `/users/{id}/events`.
- *docs/openapi.yaml*: OpenAPI-контракт обновлён под новые операции и параметры фильтрации.
- *internal/app/app.go*: инициализация приложения расширена новыми сервисами и MongoDB-хранилищами.
- *internal/router/handler.go*: реализованы новые HTTP-обработчики для событий и пользователей.
- *internal/router/handler_test.go*: добавлены и обновлены тесты для новых сценариев API.
- *internal/router/ogen/*: ogen-код синхронизирован с расширенным контрактом API.
- *internal/service/event.go*: сервис событий обновлён под новые операции и фильтры.
- *internal/service/user.go*: сервис пользователей расширен поиском и выдачей профиля.
- *internal/storage/mongodb/event.go*: хранение событий обновлено под sharding-ориентированную схему и новые запросы.
- *internal/storage/mongodb/user.go*: хранение пользователей обновлено под новые операции поиска.
