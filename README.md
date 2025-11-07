# Wallet Service — Golang + PostgreSQL + Docker

Простой, конкурентно-безопасный сервис управления кошельками.

Стек:
- Go 1.25+
- Docker + Docker Compose
- PostgreSQL

---

## 🚀 Запуск


### Собираем и запускаем
```bash
  docker-compose up --build
```
### Сервер: http://localhost:8080

## 🧪Тесты
- Unit-тесты:
```go test ./internal/model/ -v```

- Интеграционные (с Testcontainers):
```go test ./internal/repository/ -v -count=1```

- E2E (требуется запущенная БД на :5433):
```go test ./cmd/server/ -v```

## 📁Структура проекта

```
wallet-service/
├── cmd/server/          # точка входа
├── internal/
│   ├── handlers/        # HTTP-обработчики
│   ├── model/           # DTO
│   ├── repository/      # работа с БД
│   └── errors/          # типизированные ошибки
├── docker/
│   └── db-init/         # SQL-инициализация
├── docker-compose.yml
├── Dockerfile
├── config.env.example
└── README.md
```

## 🛡️ Конкурентность
```sql
SELECT balance FROM wallets WHERE id = $1 FOR UPDATE;
-- ... compute ...
UPDATE wallets SET balance = $1 WHERE id = $2;
```