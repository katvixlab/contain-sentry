# ContainSentry

**ContainSentry** — прототип инструмента, разрабатываемого в рамках выпускной квалификафикационной работы магистра на тему:

**«Разработка методики создания и эксплуатации безопасных контейнеров»**.

Идея проекта — предоставить воспроизводимый, формализованный и автоматизируемый способ проверки того, что контейнерные артефакты и практики их подготовки соответствуют требованиям методики безопасности на этапах жизненного цикла (с фокусом на ранних этапах, где нарушения дешевле устранять).

---

## Концепция и связь с методологией

ContainSentry реализует прикладную часть методики: переводит требования безопасности из текстового вида в набор формализованных проверок, которые могут применяться в инженерном процессе (локально и в CI/CD).

Инструмент ориентирован на следующие принципы:

1. **Воспроизводимость проверок**  
   Один и тот же набор требований должен одинаково интерпретироваться и выполняться в разных проектах и окружениях.

2. **Формализация “риск → контроль → рекомендация”**  
   Каждое нарушение фиксируется как отклонение от контроля, которому сопоставлены риск и корректирующее действие.

3. **Трассируемость результатов**  
   Результаты должны быть привязаны к исходным артефактам (файлам и строкам), чтобы сокращать время анализа и исправления.

4. **Приоритизация**  
   Отклонения классифицируются по критичности, чтобы поддерживать режим “gating-контроля” для обязательных требований.

5. **Расширяемость**  
   Архитектура предполагает добавление новых контролей и источников данных без изменения уже реализованных правил.

---

## Архитектура (логическая)

ContainSentry построен как конвейер:

1. **Collectors (сбор и разбор артефактов)**  
   Извлечение структуры и метаданных из входных файлов (например, Dockerfile), включая позиции строк и контекст.

2. **IR (внутреннее представление)**  
   Нормализованная модель (например, `Stage`, `Instruction`, `Span`), независимая от конкретной библиотеки парсинга.

3. **Policy Engine (движок правил)**  
   Набор формализованных проверок (политик), каждая из которых возвращает нарушения с классификацией критичности.

4. **Reporting (отчётность)**  
   Генерация человекочитаемых и машинно-обрабатываемых отчётов для ревью и интеграций.

---

## Установка

### Требования
- Go ≥ 1.22 (рекомендуется актуальная версия).

---

## Сборка из исходников

```bash
git clone github.com/katvixlab/contain-sentry
cd contain-sentry
go build -o containsentry ./cmd/containsentry
./containsentry
```

## Поддерживаемые target

ContainSentry поддерживает два домена анализа:

- `dockerfile` — анализ Dockerfile и build-практик
- `compose` — анализ Docker Compose-конфигурации

Выбор домена выполняется через `TARGET`.

## Ключи конфигурации

Сейчас запуск настраивается через переменные окружения.

| Ключ | Значение по умолчанию | Назначение |
|---|---|---|
| `TARGET` | `dockerfile` | Целевой домен: `dockerfile` или `compose` |
| `DOCKERFILE_PATH` | `Dockerfile` | Путь к Dockerfile для `TARGET=dockerfile` |
| `COMPOSE_FILES` | `compose.yaml` | Один или несколько Compose-файлов через запятую для `TARGET=compose` |
| `RULES_PATH` | `dockerfile-rules.json` | Путь к JSON-файлу с правилами |
| `REPORT_JSON` | - | Путь к JSON-отчёту с найденными замечаниями |

Примечания:

- `COMPOSE_FILES` поддерживает несколько файлов: `compose.yaml,compose.prod.yaml`
- для Compose имеет смысл использовать отдельный rules-файл, например `compose-rules.json`
- при `TARGET=dockerfile` ключ `COMPOSE_FILES` игнорируется
- при `TARGET=compose` ключ `DOCKERFILE_PATH` игнорируется

## Формат правил

Инструмент загружает правила из JSON. Поддерживаются оба формата:

- массив правил
- объект вида `{ "rules": [...] }`

Общая структура правила:

```json
{
  "target": "compose",
  "phase": "post",
  "subject": "user",
  "metadata": {
    "id": "CP002",
    "name": "Service runs as root",
    "severity": "fail"
  },
  "expression": {
    "expr_kind": "field",
    "select": "service.user",
    "expr": {
      "op": "regex",
      "pattern": "(?i)^(root|0)(?::.*)?$"
    }
  }
}
```

Поддерживаемые `target`:

- `dockerfile`
- `compose`

## Способы использования

### Просмотр справки

```bash
./containsentry --help
```

### Анализ Dockerfile

```bash
./containsentry \
  --target dockerfile \
  --dockerfile ./Dockerfile \
  --rules ./dockerfile-rules.json
```

### Анализ Docker Compose

```bash
./containsentry \
  --target compose \
  --compose-files ./compose.yaml \
  --rules ./compose-rules.json
```

### Анализ с сохранением JSON-отчёта

```bash
./containsentry \
  --target compose \
  --compose-files ./compose.yaml \
  --rules ./compose-rules.json \
  --report-json ./report.json
```

### Анализ Compose с override-файлами

```bash
./containsentry \
  --target compose \
  --compose-files ./compose.yaml,./compose.prod.yaml \
  --rules ./compose-rules.json
```

### Запуск без предварительной сборки

```bash
go run ./cmd/containsentry --help
```

```bash
go run ./cmd/containsentry \
  --target compose \
  --compose-files ./compose.yaml \
  --rules ./compose-rules.json
```

## Compose-правила

Для `compose` анализируется нормализованная конфигурация проекта, а не сырой YAML. Базовый доменный контекст включает:

- имя проекта и список файлов
- `services`
- top-level `secrets`
- `networks`
- `volumes`
- `profiles`

Поддерживаемые `subject` для `compose`:

- `service`
- `build`
- `image`
- `user`
- `read_only`
- `privileged`
- `cap_drop`
- `security_opt`
- `network_mode`
- `pid`
- `ipc`
- `devices`
- `ports`
- `volumes`
- `environment`
- `secrets`
- `healthcheck`
- `depends_on`
- `restart`
- `profiles`
- `eof`

Для Compose используется field-based DSL:

- `expr_kind: "field"`
- `select` — путь к данным, например `service.user` или `service.healthcheck`
- `expr` — булево выражение

Поддерживаемые операции:

- `exists`
- `eq`
- `ne`
- `contains`
- `in`
- `regex`
- `all`
- `any`
- `not`

## JSON Report

ContainSentry умеет сохранять результат анализа в JSON-файл.

Вывод отчёта включается через:

- CLI-флаг `--report-json ./report.json`
- переменную окружения `REPORT_JSON=./report.json`

Структура отчёта включает:

- `findings`
- `summary`

Для каждого finding в отчёт попадают:

- `id`
- `name`
- `severity`
- `description`
- `mitigation`
- `reference`
- `code_sample`
- `location`
- `target`
- `subject`

## Тестирование

```bash
go test ./...
go build ./...
```
