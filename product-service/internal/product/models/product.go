package models

import (
	"time"
)

// ProductType определяет тип продукта (например, билет, товар).
type ProductType string

const (
	Ticket      ProductType = "TICKET"
	Merchandise ProductType = "MERCHANDISE"
	// Можно добавить другие типы: Food, Drink, etc.
)

// Product представляет модель продукта (товара или билета фестиваля).
type Product struct {
	ID          int         `json:"id"`          // Уникальный идентификатор продукта (автоинкрементное число)
	Name        string      `json:"name"`        // Название продукта (например, "VIP Ticket", "Festival T-Shirt")
	Description string      `json:"description"` // Описание продукта
	Price       float64     `json:"price"`       // Цена продукта
	Type        ProductType `json:"type"`        // Тип продукта (TICKET, MERCHANDISE)
	Stock       int         `json:"stock"`       // Количество на складе (актуально для Merchandise, может быть 1 для уникальных билетов или -1 для неограниченных)
	FestivalID  *int        `json:"festival_id"` // ID фестиваля, к которому относится продукт (если применимо)
	CreatedAt   time.Time   `json:"created_at"`  // Время создания записи
	UpdatedAt   time.Time   `json:"updated_at"`  // Время последнего обновления записи
}
