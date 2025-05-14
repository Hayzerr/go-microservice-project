package repository

import (
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
)

// Repository представляет интерфейс для работы с хранилищем заказов
type Repository interface {
	// GetOrCreateCart получает или создает корзину для пользователя
	GetOrCreateCart(userID string) (*models.Order, error)

	// AddItemToCart добавляет товар в корзину
	AddItemToCart(orderID string, productID int, quantity int) (*models.OrderItem, error)

	// RemoveItemFromCart удаляет товар из корзины
	RemoveItemFromCart(orderID string, productID int) error

	// GetCartItems получает список товаров в корзине
	GetCartItems(orderID string) ([]*models.OrderItem, error)

	// GetCartByUserID получает корзину пользователя по его ID
	GetCartByUserID(userID string) (*models.Order, error)

	// CheckoutCart выполняет оформление заказа
	CheckoutCart(orderID string) error

	// GetCompletedOrders получает список выполненных заказов пользователя
	GetCompletedOrders(userID string) ([]*models.Order, error)
}
