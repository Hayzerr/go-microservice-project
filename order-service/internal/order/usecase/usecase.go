package usecase

import (
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
)

// UseCase представляет интерфейс бизнес-логики для работы с заказами
type UseCase interface {
	// AddToCart добавляет товар в корзину пользователя
	AddToCart(userID string, productID int, quantity int) error

	// RemoveFromCart удаляет товар из корзины пользователя
	RemoveFromCart(userID string, productID int) error

	// GetCart получает содержимое корзины пользователя
	GetCart(userID string) (*models.Cart, error)

	// Checkout оформляет заказ пользователя
	Checkout(userID string) error

	// GetCompletedOrders получает список выполненных заказов пользователя
	GetCompletedOrders(userID string) ([]*models.Order, error)
}
