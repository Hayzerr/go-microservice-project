package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	// ВАЖНО: Замените 'your_order_module_path' на имя вашего модуля order-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"

	"github.com/google/uuid"
	"github.com/lib/pq" // Для работы с массивами PostgreSQL, если потребуется
)

// OrderRepository определяет интерфейс для взаимодействия с хранилищем данных заказов.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) (*models.Order, error)
	// TODO: Добавить другие методы по мере необходимости
}

type postgresOrderRepository struct {
	db *sql.DB
}

// NewPostgresOrderRepository создает новый экземпляр postgresOrderRepository.
func NewPostgresOrderRepository(db *sql.DB) OrderRepository {
	return &postgresOrderRepository{db: db}
}

// CreateOrder создает новый заказ и его позиции в базе данных.
// Это должно выполняться в транзакции.
func (r *postgresOrderRepository) CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error) {
	order.ID = uuid.NewString()
	order.CreatedAt = time.Now().UTC()
	order.UpdatedAt = time.Now().UTC()
	if order.Status == "" {
		order.Status = models.StatusPending // Статус по умолчанию
	}

	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Откатываем, если что-то пошло не так

	// 1. Вставляем сам заказ
	// Предполагается таблица 'orders' со столбцами: id, user_id, total_price, status, created_at, updated_at
	orderQuery := `INSERT INTO orders (id, user_id, total_price, status, created_at, updated_at)
                   VALUES ($1, $2, $3, $4, $5, $6)
                   RETURNING id, created_at, updated_at`
	err = tx.QueryRowContext(ctx, orderQuery,
		order.ID, order.UserID, order.TotalPrice, order.Status, order.CreatedAt, order.UpdatedAt,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// 2. Вставляем позиции заказа
	// Предполагается таблица 'order_items' со столбцами: id, order_id, product_id, quantity, price
	itemQuery := `INSERT INTO order_items (id, order_id, product_id, quantity, price)
                  VALUES ($1, $2, $3, $4, $5)`
	for i := range order.Items {
		order.Items[i].ID = uuid.NewString() // Генерируем ID для каждой позиции
		order.Items[i].OrderID = order.ID     // Связываем с заказом

		_, err = tx.ExecContext(ctx, itemQuery,
			order.Items[i].ID, order.Items[i].OrderID, order.Items[i].ProductID, order.Items[i].Quantity, order.Items[i].Price,
		)
		if err != nil {
			// TODO: Обработка ошибок, например, если product_id не существует (foreign key constraint)
			return nil, err
		}
	}

	// Если все успешно, коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrderByID извлекает заказ и его позиции по ID заказа.
func (r *postgresOrderRepository) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	order := &models.Order{}
	orderQuery := `SELECT id, user_id, total_price, status, created_at, updated_at
                   FROM orders WHERE id = $1`

	err := r.db.QueryRowContext(ctx, orderQuery, orderID).Scan(
		&order.ID, &order.UserID, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Заказ не найден
		}
		return nil, err
	}

	// Загружаем позиции заказа
	itemsQuery := `SELECT id, order_id, product_id, quantity, price
                   FROM order_items WHERE order_id = $1`
	rows, err := r.db.QueryContext(ctx, itemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.OrderItem, 0)
	for rows.Next() {
		item := models.OrderItem{}
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// GetOrdersByUserID извлекает все заказы для указанного пользователя.
// Для простоты не загружает позиции заказа, но можно добавить.
func (r *postgresOrderRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	query := `SELECT id, user_id, total_price, status, created_at, updated_at
              FROM orders WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*models.Order, 0)
	for rows.Next() {
		order := &models.Order{}
		if err := rows.Scan(
			&order.ID, &order.UserID, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Для получения полного заказа с позициями, нужно будет вызвать GetOrderByID для каждого
		// или написать более сложный JOIN-запрос.
		// Здесь для примера возвращаем только основную информацию о заказе.
		// Чтобы получить позиции, можно сделать так:
		// detailedOrder, err := r.GetOrderByID(ctx, order.ID)
		// if err != nil { /* обработка ошибки */ }
		// orders = append(orders, detailedOrder)
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// UpdateOrderStatus обновляет статус существующего заказа.
func (r *postgresOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) (*models.Order, error) {
	updatedAt := time.Now().UTC()
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3
              RETURNING id, user_id, total_price, status, created_at, updated_at`

	order := &models.Order{} // Для сканирования обновленного заказа
	err := r.db.QueryRowContext(ctx, query, status, updatedAt, orderID).Scan(
		&order.ID, &order.UserID, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Заказ не найден
		}
		return nil, err
	}
	// Загружаем позиции, если нужно вернуть полный заказ
	// (или GetOrderByID можно вызвать в usecase)
	return r.GetOrderByID(ctx, order.ID) // Возвращаем полный заказ с позициями
}


// Пример схемы таблиц 'orders' и 'order_items' для PostgreSQL:
/*
CREATE TYPE order_status_enum AS ENUM (
    'PENDING', 'PAID', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'COMPLETED', 'CANCELLED', 'FAILED'
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL, -- Может ссылаться на users.id из user-service, но здесь это просто UUID
    total_price NUMERIC(10, 2) NOT NULL CHECK (total_price >= 0),
    status order_status_enum NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL, -- ID продукта (из ProductService)
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0) -- Цена за единицу на момент заказа
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);
*/
