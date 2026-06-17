# simvec

HTTP-сервис поиска похожих объектов по векторным эмбеддингам и скалярным признакам.

## Что делает

Сервис принимает произвольные объекты, векторизует их через провайдер эмбеддингов, хранит в ClickHouse и по запросу возвращает список наиболее похожих объектов из той же коллекции.

Схожесть вычисляется как взвешенная сумма двух компонентов:

- **косинусное сходство** векторов эмбеддингов;
- **скалярная схожесть** числовых признаков (нормализованных в [0, 1] по заданным диапазонам).

Веса компонентов можно задавать в каждом запросе — или не задавать, тогда все компоненты получают равный вес.

Поиск работает в два этапа: сначала ANN-поиск по вектору отбирает кандидатов (×10 от лимита, минимум 100), затем по кандидатам вычисляется итоговая оценка и результат обрезается до нужного лимита.

## Провайдеры эмбеддингов

| Тип | Описание |
|---|---|
| `local` | Пользовательский HTTP-сервис эмбеддингов — может реализовывать их самостоятельно или проксировать запросы к внешней модели |
| `gigachat/v1` | GigaChat Embeddings API |

Провайдер и его учётные данные задаются на уровне коллекции при её создании.

## API

### POST /collections

Создать коллекцию.

```json
{
  "name": "news",
  "embedderType": "gigachat/v1",
  "embedderModel": "Embeddings",
  "embedderCredentials": { "token": "<base64>" },
  "schema": [
    { "name": "title",    "type": "string" },
    { "name": "category", "type": "string" },
    { "name": "views",    "type": "number" }
  ],
  "minMaxRules": {
    "views": { "min": 0, "max": 1000000 }
  }
}
```

Поддерживаемые типы полей схемы: `string`, `number`, `bool`.  
Числовые поля (`number`) требуют явного указания диапазона в `minMaxRules`.  
Хотя бы одно поле должно быть строковым — оно используется для построения эмбеддинга.

### POST /collections/{name}/records

Загрузить объекты пакетом. Максимальный размер пакета задаётся переменной `MAX_RECORDS_PER_BATCH` (по умолчанию 1000).

```json
[
  {
    "id": "abc123",
    "data": {
      "title": "Учёные открыли новый вид",
      "category": "наука",
      "views": 42000
    }
  }
]
```

### GET /collections/{name}/records?similar-to={id}[&limit=N][&w[field]=0.5]

Найти похожие объекты.

| Параметр | По умолчанию | Описание |
|---|---|---|
| `similar-to` | обязателен | ID объекта-запроса |
| `limit` | 10 | Максимальное число результатов |
| `w[<field>]` | равные веса | Вес поля в итоговой оценке (`w[vector]`, `w[views]`, …) |

```json
{
  "results": [
    { "id": "xyz789", "score": 0.94 },
    { "id": "def456", "score": 0.87 }
  ]
}
```

### GET /ping

Healthcheck.

## Конфигурация (переменные окружения)

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_ADDR` | `:8080` | Адрес и порт HTTP-сервера |
| `MAX_RECORDS_PER_BATCH` | `1000` | Лимит объектов в одном пакете |
| `READ_HEADER_TIMEOUT_S` | `5` | Таймаут чтения заголовков (с) |
| `READ_TIMEOUT_S` | `10` | Таймаут чтения запроса (с) |
| `WRITE_TIMEOUT_S` | `10` | Таймаут записи ответа (с) |
| `IDLE_TIMEOUT_S` | `60` | Keep-alive таймаут (с) |
| `SHUTDOWN_TIMEOUT_S` | `30` | Таймаут graceful shutdown (с) |
| `CLICKHOUSE_ADDR` | — | Адрес ClickHouse |
| `CLICKHOUSE_DATABASE` | — | База данных ClickHouse |
| `CLICKHOUSE_USER` | — | Пользователь ClickHouse |
| `CLICKHOUSE_PASSWORD` | — | Пароль ClickHouse |
| `CLICKHOUSE_INIT_TIMEOUT` | `60` | Таймаут ожидания готовности ClickHouse (с) |

## Структура проекта

```
cmd/core/        — точка входа
internal/
  config/        — парсинг конфигурации из env
  domain/        — доменные сущности и типы (Collection, Record, FieldType)
  service/       — бизнес-логика (создание коллекции, пакетная загрузка, поиск)
  adapters/
    handler/     — HTTP-обработчики (chi)
    clickhouse/  — репозитории ClickHouse
    embedders/   — провайдеры эмбеддингов (local, gigachat)
  lib/           — вспомогательные утилиты (логгер, middleware, ctx-tools)
test/            — интеграционные тесты
```

## Запуск

Образ [опубликован на Docker Hub](https://hub.docker.com/repository/docker/akhmy/vecrec-core/general).

Минимальный пример запуска (требуется доступный ClickHouse):

```sh
docker run -d \
  -e CLICKHOUSE_ADDR=clickhouse:9000 \
  -e CLICKHOUSE_DATABASE=default \
  -e CLICKHOUSE_USER=default \
  -e CLICKHOUSE_PASSWORD=secret \
  -p 8080:8080 \
  TODO:image
```