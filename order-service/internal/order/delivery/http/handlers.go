package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/usecase"
	"github.com/gorilla/mux"
)

// Handler представляет HTTP-обработчик для работы с заказами
type Handler struct {
	useCase usecase.UseCase
}

// NewHandler создает новый экземпляр Handler
func NewHandler(useCase usecase.UseCase) *Handler {
	return &Handler{
		useCase: useCase,
	}
}

// RegisterRoutes регистрирует маршруты для API заказов
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/cart", h.AddToCart).Methods(http.MethodPost)
	router.HandleFunc("/api/cart/{user_id}/{product_id}", h.RemoveFromCart).Methods(http.MethodDelete)
	router.HandleFunc("/api/cart/{user_id}", h.GetCart).Methods(http.MethodGet)
	router.HandleFunc("/api/cart/{user_id}/checkout", h.Checkout).Methods(http.MethodPost)
	router.HandleFunc("/api/orders/{user_id}", h.GetCompletedOrders).Methods(http.MethodGet)
}

// AddToCartRequest представляет запрос на добавление товара в корзину
type AddToCartRequest struct {
	UserID    string `json:"user_id"`
	ProductID int    `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// SuccessResponse представляет успешный ответ
type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}

// AddToCart обрабатывает запрос на добавление товара в корзину
func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	var req AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "Не указан ID пользователя", http.StatusBadRequest)
		return
	}

	if req.ProductID <= 0 {
		http.Error(w, "Некорректный ID товара", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "Количество должно быть положительным числом", http.StatusBadRequest)
		return
	}

	err := h.useCase.AddToCart(req.UserID, req.ProductID, req.Quantity)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Status:  "success",
		Message: "Товар успешно добавлен в корзину",
	})
}

// RemoveFromCart обрабатывает запрос на удаление товара из корзины
func (h *Handler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]
	productIDStr := vars["product_id"]

	if userID == "" {
		http.Error(w, "Не указан ID пользователя", http.StatusBadRequest)
		return
	}

	productID, err := strconv.Atoi(productIDStr)
	if err != nil || productID <= 0 {
		http.Error(w, "Некорректный ID товара", http.StatusBadRequest)
		return
	}

	err = h.useCase.RemoveFromCart(userID, productID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Status:  "success",
		Message: "Товар успешно удален из корзины",
	})
}

// GetCart обрабатывает запрос на получение содержимого корзины
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		http.Error(w, "Не указан ID пользователя", http.StatusBadRequest)
		return
	}

	cart, err := h.useCase.GetCart(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cart)
}

// Checkout обрабатывает запрос на оформление заказа
func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		http.Error(w, "Не указан ID пользователя", http.StatusBadRequest)
		return
	}

	err := h.useCase.Checkout(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Status:  "success",
		Message: "Заказ успешно оформлен",
	})
}

// GetCompletedOrders обрабатывает запрос на получение выполненных заказов пользователя
func (h *Handler) GetCompletedOrders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		http.Error(w, "Не указан ID пользователя", http.StatusBadRequest)
		return
	}

	orders, err := h.useCase.GetCompletedOrders(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}
