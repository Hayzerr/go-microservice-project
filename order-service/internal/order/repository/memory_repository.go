package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/google/uuid"
)

// MemoryRepository представляет репозиторий для работы с заказами, хранящимися в памяти
type MemoryRepository struct {
	orders     map[string]*models.Order       // Хранение заказов по ID
	orderItems map[string][]*models.OrderItem // Хранение товаров в заказе по ID заказа
	userOrders map[string]string              // Хранение текущей активной корзины пользователя (user_id -> order_id)
	mu         sync.RWMutex                   // Мьютекс для безопасного доступа к данным
}

// NewMemoryRepository создает новый экземпляр in-memory репозитория
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders:     make(map[string]*models.Order),
		orderItems: make(map[string][]*models.OrderItem),
		userOrders: make(map[string]string),
	}
}

// GetOrCreateCart получает или создает корзину для пользователя
func (r *MemoryRepository) GetOrCreateCart(userID string) (*models.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, есть ли у пользователя активная корзина
	if orderID, exists := r.userOrders[userID]; exists {
		if order, ok := r.orders[orderID]; ok && order.Status == models.StatusCart {
			return order, nil
		}
	}

	// Создаем новую корзину
	orderID := uuid.New().String()
	now := time.Now()
	order := &models.Order{
		ID:        orderID,
		UserID:    userID,
		Status:    models.StatusCart,
		CreatedAt: now,
		UpdatedAt: now,
	}

	r.orders[orderID] = order
	r.userOrders[userID] = orderID
	r.orderItems[orderID] = []*models.OrderItem{}

	return order, nil
}

// AddItemToCart добавляет товар в корзину
func (r *MemoryRepository) AddItemToCart(orderID string, productID int, quantity int) (*models.OrderItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем существование заказа
	order, exists := r.orders[orderID]
	if !exists {
		return nil, errors.New("заказ не найден")
	}

	if order.Status != models.StatusCart {
		return nil, errors.New("заказ уже оформлен")
	}

	// Проверяем, есть ли уже этот товар в корзине
	items := r.orderItems[orderID]
	for _, item := range items {
		if item.ProductID == productID {
			// Увеличиваем количество
			item.Quantity += quantity
			item.UpdatedAt = time.Now()
			return item, nil
		}
	}

	// Добавляем новый товар
	now := time.Now()
	item := &models.OrderItem{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  quantity,
		CreatedAt: now,
		UpdatedAt: now,
	}

	r.orderItems[orderID] = append(r.orderItems[orderID], item)
	return item, nil
}

// RemoveItemFromCart удаляет товар из корзины
func (r *MemoryRepository) RemoveItemFromCart(orderID string, productID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем существование заказа
	order, exists := r.orders[orderID]
	if !exists {
		return errors.New("заказ не найден")
	}

	if order.Status != models.StatusCart {
		return errors.New("заказ уже оформлен")
	}

	// Находим товар в корзине
	items := r.orderItems[orderID]
	for i, item := range items {
		if item.ProductID == productID {
			// Удаляем товар из списка
			r.orderItems[orderID] = append(items[:i], items[i+1:]...)
			return nil
		}
	}

	return errors.New("товар не найден в корзине")
}

// GetCartItems получает список товаров в корзине
func (r *MemoryRepository) GetCartItems(orderID string) ([]*models.OrderItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Проверяем существование заказа
	_, exists := r.orders[orderID]
	if !exists {
		return nil, errors.New("заказ не найден")
	}

	items := r.orderItems[orderID]
	// Копируем слайс, чтобы избежать ошибок с конкурентным доступом
	result := make([]*models.OrderItem, len(items))
	copy(result, items)

	return result, nil
}

// GetCartByUserID получает корзину пользователя по его ID
func (r *MemoryRepository) GetCartByUserID(userID string) (*models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orderID, exists := r.userOrders[userID]
	if !exists {
		return nil, errors.New("корзина не найдена")
	}

	order, exists := r.orders[orderID]
	if !exists {
		return nil, errors.New("заказ не найден")
	}

	if order.Status != models.StatusCart {
		return nil, errors.New("нет активной корзины")
	}

	return order, nil
}

// CheckoutCart выполняет оформление заказа
func (r *MemoryRepository) CheckoutCart(orderID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем существование заказа
	order, exists := r.orders[orderID]
	if !exists {
		return errors.New("заказ не найден")
	}

	if order.Status != models.StatusCart {
		return errors.New("заказ уже оформлен")
	}

	// Проверяем, что в корзине есть товары
	items := r.orderItems[orderID]
	if len(items) == 0 {
		return errors.New("корзина пуста")
	}

	// Обновляем статус заказа
	order.Status = models.StatusCheckout
	order.UpdatedAt = time.Now()

	return nil
}

// GetCompletedOrders получает список выполненных заказов пользователя
func (r *MemoryRepository) GetCompletedOrders(userID string) ([]*models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var completedOrders []*models.Order

	// Перебираем все заказы и находим выполненные заказы пользователя
	for _, order := range r.orders {
		if order.UserID == userID && order.Status == models.StatusCheckout {
			// Создаем копию заказа, чтобы избежать проблем с конкурентным доступом
			orderCopy := *order
			completedOrders = append(completedOrders, &orderCopy)
		}
	}

	if len(completedOrders) == 0 {
		return nil, errors.New("выполненные заказы не найдены")
	}

	return completedOrders, nil
}
