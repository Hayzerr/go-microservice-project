package usecase

import (
	"errors"
	"fmt"

	"github.com/Hayzerr/go-microservice-project/order-service/internal/clients"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/repository"
)

// OrderUseCase представляет реализацию интерфейса UseCase
type OrderUseCase struct {
	repo          repository.Repository
	userClient    *clients.UserClient
	productClient *clients.ProductClient
}

// NewOrderUseCase создает новый экземпляр OrderUseCase
func NewOrderUseCase(repo repository.Repository, userClient *clients.UserClient, productClient *clients.ProductClient) *OrderUseCase {
	return &OrderUseCase{
		repo:          repo,
		userClient:    userClient,
		productClient: productClient,
	}
}

// AddToCart добавляет товар в корзину пользователя
func (u *OrderUseCase) AddToCart(userID string, productID int, quantity int) error {
	if quantity <= 0 {
		return errors.New("количество должно быть положительным числом")
	}

	// Проверяем существование пользователя
	user, err := u.userClient.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("ошибка проверки пользователя: %w", err)
	}
	if user == nil {
		return errors.New("пользователь не найден")
	}

	// Проверяем существование товара
	product, err := u.productClient.GetProductByID(productID)
	if err != nil {
		return fmt.Errorf("ошибка проверки товара: %w", err)
	}
	if product == nil {
		return errors.New("товар не найден")
	}

	// Проверяем наличие товара на складе
	if product.Stock != -1 && product.Stock < quantity {
		return fmt.Errorf("недостаточное количество товара на складе (доступно: %d)", product.Stock)
	}

	// Получаем или создаем корзину пользователя
	cart, err := u.repo.GetOrCreateCart(userID)
	if err != nil {
		return fmt.Errorf("ошибка получения корзины: %w", err)
	}

	// Добавляем товар в корзину
	_, err = u.repo.AddItemToCart(cart.ID, productID, quantity)
	if err != nil {
		return fmt.Errorf("ошибка добавления товара в корзину: %w", err)
	}

	return nil
}

// RemoveFromCart удаляет товар из корзины пользователя
func (u *OrderUseCase) RemoveFromCart(userID string, productID int) error {
	// Получаем корзину пользователя
	cart, err := u.repo.GetCartByUserID(userID)
	if err != nil {
		return fmt.Errorf("ошибка получения корзины: %w", err)
	}

	// Удаляем товар из корзины
	err = u.repo.RemoveItemFromCart(cart.ID, productID)
	if err != nil {
		return fmt.Errorf("ошибка удаления товара из корзины: %w", err)
	}

	return nil
}

// GetCart получает содержимое корзины пользователя
func (u *OrderUseCase) GetCart(userID string) (*models.Cart, error) {
	// Получаем корзину пользователя
	cart, err := u.repo.GetCartByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения корзины: %w", err)
	}

	// Получаем товары в корзине
	items, err := u.repo.GetCartItems(cart.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения товаров из корзины: %w", err)
	}

	// Собираем полную информацию о корзине
	result := &models.Cart{
		Order:      *cart,
		Items:      make([]models.CartItem, 0, len(items)),
		TotalPrice: 0,
	}

	for _, item := range items {
		// Получаем информацию о товаре
		product, err := u.productClient.GetProductByID(item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения информации о товаре %d: %w", item.ProductID, err)
		}

		totalPrice := product.Price * float64(item.Quantity)
		result.Items = append(result.Items, models.CartItem{
			OrderItem:    *item,
			ProductName:  product.Name,
			ProductPrice: product.Price,
			TotalPrice:   totalPrice,
		})

		result.TotalPrice += totalPrice
	}

	return result, nil
}

// Checkout оформляет заказ пользователя
func (u *OrderUseCase) Checkout(userID string) error {
	// Получаем корзину пользователя
	cart, err := u.repo.GetCartByUserID(userID)
	if err != nil {
		return fmt.Errorf("ошибка получения корзины: %w", err)
	}

	// Оформляем заказ
	err = u.repo.CheckoutCart(cart.ID)
	if err != nil {
		return fmt.Errorf("ошибка оформления заказа: %w", err)
	}

	return nil
}

// GetCompletedOrders получает список выполненных заказов пользователя
func (u *OrderUseCase) GetCompletedOrders(userID string) ([]*models.Order, error) {
	// Проверяем существование пользователя
	user, err := u.userClient.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки пользователя: %w", err)
	}
	if user == nil {
		return nil, errors.New("пользователь не найден")
	}

	// Получаем выполненные заказы пользователя
	orders, err := u.repo.GetCompletedOrders(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения выполненных заказов: %w", err)
	}

	return orders, nil
}
