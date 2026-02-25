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
