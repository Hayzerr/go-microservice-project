# Order Service

Order Service - микросервис для управления корзиной и заказами пользователей в рамках микросервисной архитектуры.

## Структура проекта

```
order-service/
  ├── internal/
  │   ├── clients/                # Клиенты для взаимодействия с другими сервисами
  │   │   ├── product_client.go   # Клиент для product-service
  │   │   └── user_client.go      # Клиент для user-service
  │   └── order/                  # Основной модуль заказов
  │       ├── delivery/           # Слой доставки (API, gRPC)
  │       │   └── http/           # HTTP API
  │       ├── models/             # Модели данных
  │       ├── repository/         # Слой хранения данных
  │       └── usecase/            # Бизнес-логика
  ├── main.go                     # Точка входа
  ├── go.mod                      # Go модуль
  └── Dockerfile                  # Dockerfile для контейнеризации
```

## API Endpoints

### 1. Добавить товар в корзину

```
POST /api/cart
```

#### Запрос:

```json
{
  "user_id": "user123",
  "product_id": 42,
  "quantity": 2
}
```

#### Ответ (успех):

```json
{
  "status": "success",
  "message": "Товар успешно добавлен в корзину"
}
```

#### Ответ (ошибка):

```json
{
  "error": "пользователь не найден"
}
```

### 2. Удалить товар из корзины

```
DELETE /api/cart/{user_id}/{product_id}
```

#### Ответ (успех):

```json
{
  "status": "success",
  "message": "Товар успешно удален из корзины"
}
```

#### Ответ (ошибка):

```json
{
  "error": "товар не найден в корзине"
}
```

### 3. Получить содержимое корзины

```
GET /api/cart/{user_id}
```

#### Ответ (успех):

```json
{
  "id": "order123",
  "user_id": "user123",
  "status": "CART",
  "created_at": "2023-09-20T15:30:45Z",
  "updated_at": "2023-09-20T15:30:45Z",
  "items": [
    {
      "id": "item123",
      "order_id": "order123",
      "product_id": 42,
      "quantity": 2,
      "created_at": "2023-09-20T15:30:45Z",
      "updated_at": "2023-09-20T15:30:45Z",
      "product_name": "Футболка",
      "product_price": 1500.0,
      "total_price": 3000.0
    }
  ],
  "total_price": 3000.0
}
```

#### Ответ (ошибка):

```json
{
  "error": "корзина не найдена"
}
```

### 4. Оформить заказ

```
POST /api/cart/{user_id}/checkout
```

#### Ответ (успех):

```json
{
  "status": "success",
  "message": "Заказ успешно оформлен"
}
```

#### Ответ (ошибка):

```json
{
  "error": "корзина пуста"
}
```

### 5. Получить выполненные заказы пользователя

```
GET /api/orders/{user_id}
```

#### Ответ (успех):

```json
[
  {
    "id": "order123",
    "user_id": "user123",
    "status": "CHECKOUT",
    "created_at": "2023-09-20T15:30:45Z",
    "updated_at": "2023-09-20T16:45:12Z"
  },
  {
    "id": "order456",
    "user_id": "user123",
    "status": "CHECKOUT",
    "created_at": "2023-09-22T10:15:30Z",
    "updated_at": "2023-09-22T10:20:18Z"
  }
]
```

#### Ответ (ошибка):

```json
{
  "error": "выполненные заказы не найдены"
}
```

## Переменные окружения

- `HTTP_PORT` - порт для HTTP сервера (по умолчанию "8083")
- `GRPC_PORT` - порт для gRPC сервера (по умолчанию "50053")
- `USER_SERVICE_URL` - URL для user-service (по умолчанию "http://localhost:8081")
- `PRODUCT_SERVICE_URL` - URL для product-service (по умолчанию "http://localhost:8082")
- `MOCK_SERVICES` - если установлено в "true", использует моковые данные вместо реальных сервисов (полезно для тестирования)

## Моковый режим

Сервис автоматически определяет доступность user-service и product-service при запуске. Если какой-либо из этих сервисов недоступен, автоматически включается моковый режим, который позволяет тестировать функциональность order-service без запуска других сервисов.

Вы также можете вручную включить моковый режим, установив переменную окружения:

```
MOCK_SERVICES=true go run main.go
```

## Запуск сервиса

```
go run main.go
```

## Запуск через Docker

```
docker build -t order-service .
docker run -p 8083:8083 -p 50053:50053 order-service
```

## Примеры API запросов

### Добавление товара в корзину
```
curl -X POST http://localhost:8083/api/cart \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "product_id": 42,
    "quantity": 2
  }'
```

### Получение содержимого корзины
```
curl -X GET http://localhost:8083/api/cart/user123
```

### Удаление товара из корзины
```
curl -X DELETE http://localhost:8083/api/cart/user123/42
```

### Оформление заказа
```
curl -X POST http://localhost:8083/api/cart/user123/checkout
```

### Получение выполненных заказов
```
curl -X GET http://localhost:8083/api/orders/user123
``` 