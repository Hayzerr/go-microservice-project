package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	// ВАЖНО: Замените 'your_product_module_path' на имя вашего модуля product-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"

	"github.com/google/uuid"
)

// ProductRepository определяет интерфейс для взаимодействия с хранилищем данных продуктов.
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) (*models.Product, error)
	GetByID(ctx context.Context, id string) (*models.Product, error)
	ListAll(ctx context.Context) ([]*models.Product, error) // Метод для получения списка продуктов
	Update(ctx context.Context, product *models.Product) (*models.Product, error)
	Delete(ctx context.Context, id string) error
	// Можно добавить методы для фильтрации, например, ListByFestivalID, ListByType
}

// postgresProductRepository реализует ProductRepository для PostgreSQL.
type postgresProductRepository struct {
	db *sql.DB
}

// NewPostgresProductRepository создает новый экземпляр postgresProductRepository.
func NewPostgresProductRepository(db *sql.DB) ProductRepository {
	return &postgresProductRepository{db: db}
}

// Create создает новую запись продукта в базе данных.
func (r *postgresProductRepository) Create(ctx context.Context, product *models.Product) (*models.Product, error) {
	product.ID = uuid.NewString()
	product.CreatedAt = time.Now().UTC()
	product.UpdatedAt = time.Now().UTC()

	// Предполагается таблица 'products' со столбцами:
	// id, name, description, price, type, stock, festival_id, created_at, updated_at
	query := `INSERT INTO products (id, name, description, price, type, stock, festival_id, created_at, updated_at)
			   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			   RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		product.ID, product.Name, product.Description, product.Price, product.Type, product.Stock, product.FestivalID, product.CreatedAt, product.UpdatedAt,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		// TODO: Обработка специфичных ошибок БД (например, нарушение ограничений)
		return nil, err
	}
	return product, nil
}

// GetByID извлекает продукт из базы данных по его ID.
func (r *postgresProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	product := &models.Product{}
	query := `SELECT id, name, description, price, type, stock, festival_id, created_at, updated_at
			   FROM products
			   WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price, &product.Type, &product.Stock, &product.FestivalID, &product.CreatedAt, &product.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Продукт не найден
		}
		return nil, err
	}
	return product, nil
}

// ListAll извлекает все продукты из базы данных.
// В реальном приложении здесь, скорее всего, понадобится пагинация.
func (r *postgresProductRepository) ListAll(ctx context.Context) ([]*models.Product, error) {
	query := `SELECT id, name, description, price, type, stock, festival_id, created_at, updated_at
			   FROM products ORDER BY created_at DESC` // Пример сортировки

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]*models.Product, 0)
	for rows.Next() {
		product := &models.Product{}
		if err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Type, &product.Stock, &product.FestivalID, &product.CreatedAt, &product.UpdatedAt,
		); err != nil {
			return nil, err // Ошибка при сканировании строки
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err // Ошибка после итерации
	}

	return products, nil
}

// Update обновляет существующую запись продукта в базе данных.
func (r *postgresProductRepository) Update(ctx context.Context, product *models.Product) (*models.Product, error) {
	product.UpdatedAt = time.Now().UTC()
	query := `UPDATE products
			   SET name = $1, description = $2, price = $3, type = $4, stock = $5, festival_id = $6, updated_at = $7
			   WHERE id = $8
			   RETURNING id, name, description, price, type, stock, festival_id, created_at, updated_at`

	updatedProduct := &models.Product{}
	err := r.db.QueryRowContext(ctx, query,
		product.Name, product.Description, product.Price, product.Type, product.Stock, product.FestivalID, product.UpdatedAt, product.ID,
	).Scan(
		&updatedProduct.ID, &updatedProduct.Name, &updatedProduct.Description, &updatedProduct.Price, &updatedProduct.Type, &updatedProduct.Stock, &updatedProduct.FestivalID, &updatedProduct.CreatedAt, &updatedProduct.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Продукт не найден для обновления
		}
		return nil, err
	}
	return updatedProduct, nil
}

// Delete удаляет продукт из базы данных по его ID.
func (r *postgresProductRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Продукт не найден для удаления
	}
	return nil
}

// Пример схемы таблицы 'products' для PostgreSQL:
/*
CREATE TYPE product_type AS ENUM ('TICKET', 'MERCHANDISE'); -- Пример создания ENUM типа

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0), -- Цена с 2 знаками после запятой, не отрицательная
    type product_type NOT NULL, -- Использование ENUM типа
    stock INT NOT NULL DEFAULT 0, -- Количество на складе
    festival_id UUID, -- ID фестиваля, может быть NULL или ссылаться на таблицу фестивалей
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
*/
