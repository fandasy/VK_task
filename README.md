# PubSub Service

# Описание
Реализация сервиса подписок на основе паттерна Publisher-Subscriber с использованием gRPC.

# Содержание

- [Архитектура](#архитектура)
- [Компоненты](#компоненты)
    - [SubPub package](#1-SubPub-package)
    - [gRPC Server API](#2-grpc-server-api)
- [Запуск](#запуск)
    - [Config](#config)
    - [Ручной запуск](#ручной-запуск)
    - [Docker](#docker-compose)
- [Паттерны](#использованные-паттерны)
  - [Фасад](#фасад)

# Архитектура

```text
📦 pubsub-app
├───🚀 cmd
│   └───pubsub-server/main.go                   
│
├───⚙️ configs               # Конфигурация окружений
│       dev.yaml             # - разработка
│       local.yaml           # - локальная
│       prod.yaml            # - продакшен
│
├───🔒 internal
│   ├───🏗️ app               # Инициализация приложения
│   │   └───grpc             # инициализация gRPC-Server
│   │
│   ├───📝 config
│   │
│   ├───🌐 grpc              # gRPC транспорт
│   │   ├───🛠️ handler
│   │   └───🔌 middleware
│   │
│   ├───📦 pkg
│   │   └───📋 logger
│   │       └───sl           # Вспомогательные методы для slog
│   │
│   └───🧪 tests             # Тесты приложения
│
├───📚 pkg
│   ├───📡 api
│   │   └───pubsub           # Сгенерированные protobuf файлы
│   │
│   ├───❗ e                  # Вспомогательные методы для error
│   │
│   └───🔄 subpub            # Пакет SubPub
│       └──🧪 subpub_test    # Тесты пакета
│
└───📜 protoс                # Прото-контракты
    │   Makefile             # Скрипты генерации
    ├──⚡ gen                # Сгенерированные protobuf файлы
    └──📄 proto              # Исходные .proto файлы
```

# Компоненты

## 1. SubPub package
- **Реализация:** [pkg/subpub](./pkg/subpub)
- **Тесты:** [pkg/subpub/subpub_test](./pkg/subpub/subpub_test/subpub_test.go)

### API

```go
type MessageHandler func(msg interface{})

type Subscription interface {
    Unsubscribe()
}

type SubPub interface {
    Subscribe(subject string, cb MessageHandler) (Subscription, error)
    Publish(subject string, msg interface{}) error
    Close(ctx context.Context) error
}
```

### MessageHandler

>**Тип** `func(msg interface{})` - Функция обратного вызова, которая:
>- Вызывается для каждого нового сообщения в подписке
>- Не начнёт обработку следующего сообщения, пока текущее не будет обработано

### Subscription

>**Метод** `Unsubscribe`, действие:
>- Удаляет подписчика из указанного subject
>- Если subject остаётся без подписчиков - полностью удаляет subject

### SubPub

>**Метод** `Subscribe`, действие:
>- Создаёт новый subject, если он не существует
>- Регистрирует подписчика с callback-функцией
>
>Ошибки:
`ErrInvalidArgument` | `ErrSubPubClosed`

>**Метод** `Publish`, действие:
>- Отправляет сообщение в очередь subject
>
>Ошибки:
`ErrInvalidArgument` | `ErrNoSuchSubject` | `ErrSubPubClosed`

>**Метод** `Close`, действие:
>- Прекращает приём новых запросов
>- Закрывает все subject и subscription
>- Ожидает завершения:
>  - Всех обработчиков MessageHandler
>  - Либо истечения context
>
>Ошибки:
`ErrSubPubClosed`

## 2. gRPC Server API
- **Реализация:** [internal/grpc/handler/pubsub](./internal/grpc/handler/pubsub/service.go)
- **Тесты:** [internal/tests](./internal/tests/app_test.go) *(требуется запущенный сервер)*

---

### Subscribe (Stream)

**Параметры:**
- `key` (string) - название subject, *required*

**Возвращает:**
`stream Event` где:
```protobuf
message Event {
  string data = 1;
}
```

**Возможные ошибки:**
- `codes.InvalidArgument` - key required
- `codes.Internal` - failed to subscribe
- `codes.Unavailable` - failed to send event: `err`
- `codes.Canceled` - Server stopping

---

### Publish (Unary)

**Параметры:**
- `key` (string) - название subject, *required*
- `data` (string) - содержимое сообщения, *required*

**Возвращает:**
`google.protobuf.Empty` при успехе

**Возможные ошибки:**
- `codes.InvalidArgument` - key required
- `codes.InvalidArgument` - data required
- `codes.InvalidArgument` - no such subject
- `codes.Internal` - failed to publish

# Запуск

## Config

### Пример конфига
```yaml
slog:
  env: "dev"   # Режим логирования
  file: ""     # Файл для логов (пусто = stdout)

grpc:
  addr: "0.0.0.0"  # Интерфейс прослушивания
  port: 8082       # Порт сервера

sub_pub:
  subject_buffer: 16       # Буфер сообщений темы
  subscription_buffer: 64  # Буфер подписки
  close_timeout: 30s       # Таймаут завершения
```

### Описание параметров

#### Логирование (slog)
- **env - формат логирование**
  - `local` - текстовый формат, Debug
  - `dev`   - json, Debug
  - `prod` - json, Info
- **file**
  - Пусто: вывод в консоль
  - Указано: запись логов в файл

#### gRPC Сервер
- **addr** - Интерфейс для прослушивания
- **port** - Порт сервера

#### PubSub Настройки
- **subject_buffer** `(int)` - Размер буфера сообщений для темы (subject)
- **subscription_buffer** `(int)` - Размер буфера для подписчика
- **close_timeout** `(duration)` - Макс. время завершения обработчиков

---

## Ручной запуск

### Требования
- Go 1.24.2+
- Доступные порты (проверьте конфликты в конфигурации), по умолчанию 8082

```bash
git clone https://github.com/fandasy/VK_task.git
cd VK_task

# Запуск сервера с production-конфигом
go run ./cmd/pubsub-server/main.go --config=./config/prod.yaml
```

---

## Docker-compose

### Требования
- Docker
- Docker compose
- Доступные порты (проверьте конфликты в конфигурации), по умолчанию 8082

Проверьте соответствие портов в:
- [config/prod.yaml](./config/prod.yaml)
- [docker-compose.yaml](./docker-compose.yaml)

```bash
# Сборка и запуск контейнеров
docker-compose up --build
```

```bash
# Только запуск (без пересборки)
docker-compose up
```

# Использованные паттерны

## Фасад
[internal/app/app.go](./internal/app/app.go) — предоставляет упрощенный интерфейс для запуска и остановки всего приложения.

## TODO