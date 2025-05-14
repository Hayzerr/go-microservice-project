package models

import (
	"time"
)

// OrderStatus представляет статус заказа
type OrderStatus string

const (
	StatusCart     OrderStatus = "CART"     // Товары в корзине, заказ не оформлен
	StatusCheckout OrderStatus = "CHECKOUT" // Заказ оформлен
)

// Order представляет заказ пользователя
type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// OrderItem представляет товар в заказе
type OrderItem struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	ProductID int       `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CartItem представляет товар в корзине с деталями продукта
type CartItem struct {
	OrderItem
	ProductName  string  `json:"product_name"`
	ProductPrice float64 `json:"product_price"`
	TotalPrice   float64 `json:"total_price"`
}

// Cart представляет корзину пользователя с товарами
type Cart struct {
	Order
	Items      []CartItem `json:"items"`
	TotalPrice float64    `json:"total_price"`
}
