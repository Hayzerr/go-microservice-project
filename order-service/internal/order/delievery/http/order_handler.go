package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	// ВАЖНО: Замените 'your_order_module_path' на имя вашего модуля order-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/usecase"

	// Рекомендуется использовать более продвинутый роутер, например chi или gorilla/mux
)

// OrderHTTPHandler обрабатывает HTTP запросы, связанные с заказами.
type OrderHTTPHandler struct {
	orderUsecase usecase.OrderUsecase
}

// NewOrderHTTPHandler создает новый экземпляр OrderHTTPHandler.
func NewOrderHTTPHandler(uc usecase.OrderUsecase) *OrderHTTPHandler {
	return &OrderHTTPHandler{orderUsecase: uc}
}

// RegisterRoutes регистрирует HTTP маршруты для обработчика заказов.
func (h *OrderHTTPHandler) RegisterRoutes(router *http.ServeMux) {
	// POST /api/orders - Создать новый заказ
	// GET  /api/orders/user/{userID} - Получить все заказы пользователя
	router.HandleFunc("/api/orders", h.handleCreateOrder) // Будет только POST
	router.HandleFunc("/api/orders/user/", h.handleUserOrders) // Для /api/orders/user/{userID}

	// GET    /api/orders/{orderID} - Получить заказ по ID
	// PUT    /api/orders/{orderID}/status - Обновить статус заказа
	router.HandleFunc("/api/orders/", h.handleOrderByID)
}

// handleCreateOrder обрабатывает только POST запросы на /api/orders
func (h *OrderHTTPHandler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.createOrder(w, r)
	} else {
		http.Error(w, "Метод не разрешен для /api/orders. Используйте POST для создания.", http.StatusMethodNotAllowed)
	}
}

// handleUserOrders обрабатывает GET запросы на /api/orders/user/{userID}
func (h *OrderHTTPHandler) handleUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Извлечение userID из пути, например /api/orders/user/some-user-uuid
		userID := strings.TrimPrefix(r.URL.Path, "/api/orders/user/")
		if userID == "" || userID == r.URL.Path {
			http.Error(w, "UserID не указан в пути", http.StatusBadRequest)
			return
		}
		h.listOrdersByUserID(w, r, userID)
	} else {
		http.Error(w, "Метод не разрешен для /api/orders/user/", http.StatusMethodNotAllowed)
	}
}

// handleOrderByID обрабатывает запросы к /api/orders/{orderID} и /api/orders/{orderID}/status
func (h *OrderHTTPHandler) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/orders/")
	parts := strings.Split(path, "/") // parts[0] будет orderID, parts[1] (если есть) будет "status"

	orderID := parts[0]
	if orderID == "" {
		http.Error(w, "OrderID не указан в пути", http.StatusBadRequest)
		return
	}

	if len(parts) == 1 { // Это /api/orders/{orderID}
		switch r.Method {
		case http.MethodGet:
			h.getOrderByID(w, r, orderID)
		default:
			http.Error(w, "Метод не разрешен для /api/orders/{orderID}", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 2 && parts[1] == "status" { // Это /api/orders/{orderID}/status
		switch r.Method {
		case http.MethodPut:
			h.updateOrderStatus(w, r, orderID)
		default:
			http.Error(w, "Метод не разрешен для /api/orders/{orderID}/status", http.StatusMethodNotAllowed)
		}
	} else {
		http.NotFound(w, r)
	}
}


// createOrder обрабатывает запрос на создание нового заказа.
func (h *OrderHTTPHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Базовая валидация (более сложная валидация в usecase)
	if input.UserID == "" || len(input.Items) == 0 {
		http.Error(w, "UserID и хотя бы одна позиция заказа (items) обязательны", http.StatusBadRequest)
		return
	}
	for _, item := range input.Items {
		if item.ProductID == "" || item.Quantity <= 0 {
			http.Error(w, "ProductID и Quantity (>0) обязательны для каждой позиции", http.StatusBadRequest)
			return
		}
	}


	order, err := h.orderUsecase.CreateOrder(r.Context(), input)
	if err != nil {
		// Обработка специфичных ошибок из usecase
		switch {
		case errors.Is(err, usecase.ErrInvalidOrderInput):
			http.Error(w, "Некорректные данные заказа: "+err.Error(), http.StatusBadRequest)
		case errors.Is(err, usecase.ErrUserNotFoundForOrder):
			http.Error(w, "Пользователь для заказа не найден: "+err.Error(), http.StatusNotFound)
		case errors.Is(err, usecase.ErrProductNotFoundForOrder):
			http.Error(w, "Один или несколько продуктов не найдены: "+err.Error(), http.StatusNotFound)
		case errors.Is(err, usecase.ErrInsufficientStock):
			http.Error(w, "Недостаточно товара на складе: "+err.Error(), http.StatusConflict) // 409 Conflict
		case errors.Is(err, usecase.ErrCreateOrderFailed):
			http.Error(w, "Не удалось создать заказ: "+err.Error(), http.StatusInternalServerError)
		default:
			http.Error(w, "Внутренняя ошибка сервера при создании заказа: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// getOrderByID обрабатывает запрос на получение заказа по ID.
func (h *OrderHTTPHandler) getOrderByID(w http.ResponseWriter, r *http.Request, orderID string) {
	order, err := h.orderUsecase.GetOrderByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, usecase.ErrOrderNotFound) {
			http.Error(w, "Заказ не найден", http.StatusNotFound)
		} else {
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

// listOrdersByUserID обрабатывает запрос на получение списка заказов пользователя.
func (h *OrderHTTPHandler) listOrdersByUserID(w http.ResponseWriter, r *http.Request, userID string) {
	orders, err := h.orderUsecase.ListOrdersByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFoundForOrder) { // Если usecase возвращает эту ошибку
			http.Error(w, "Пользователь не найден: "+err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

// updateOrderStatus обрабатывает запрос на обновление статуса заказа.
func (h *OrderHTTPHandler) updateOrderStatus(w http.ResponseWriter, r *http.Request, orderID string) {
	var requestBody struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newStatus := models.OrderStatus(requestBody.Status)
	// Валидация значения статуса (можно сделать более строгой, проверив по списку допустимых)
	if newStatus == "" {
		http.Error(w, "Новый статус заказа не может быть пустым", http.StatusBadRequest)
		return
	}


	updatedOrder, err := h.orderUsecase.UpdateOrderStatus(r.Context(), orderID, newStatus)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrOrderNotFound):
			http.Error(w, "Заказ для обновления статуса не найден", http.StatusNotFound)
		case errors.Is(err, usecase.ErrInvalidOrderInput): // Если usecase возвращает эту ошибку для статуса
			http.Error(w, "Некорректный новый статус: "+err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedOrder)
}
