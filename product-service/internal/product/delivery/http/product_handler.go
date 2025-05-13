package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings" // Для извлечения ID из URL

	// ВАЖНО: Замените 'your_product_module_path' на имя вашего модуля product-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/repository"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/usecase"
	// Рекомендуется использовать более продвинутый роутер, например chi или gorilla/mux
	// import "github.com/go-chi/chi/v5"
)

// ProductHTTPHandler обрабатывает HTTP запросы, связанные с продуктами.
type ProductHTTPHandler struct {
	productUsecase usecase.ProductUsecase
	repo           repository.ProductRepository
}

// NewProductHTTPHandler создает новый экземпляр ProductHTTPHandler.
func NewProductHTTPHandler(uc usecase.ProductUsecase, repo repository.ProductRepository) *ProductHTTPHandler {
	return &ProductHTTPHandler{productUsecase: uc, repo: repo}
}

// RegisterRoutes регистрирует HTTP маршруты для обработчика продуктов.
// Этот метод адаптирован для стандартного http.ServeMux.
// При использовании роутера типа chi, регистрация будет выглядеть иначе.
func (h *ProductHTTPHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("/api/products", h.handleProducts)     // GET (list), POST (create)
	router.HandleFunc("/api/products/", h.handleProductByID) // GET (by ID), PUT (update), DELETE (by ID)
}

// handleProducts обрабатывает запросы к /api/products (список и создание)
func (h *ProductHTTPHandler) handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProducts(w, r)
	case http.MethodPost:
		h.createProduct(w, r)
	default:
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
	}
}

// handleProductByID обрабатывает запросы к /api/products/{id}
func (h *ProductHTTPHandler) handleProductByID(w http.ResponseWriter, r *http.Request) {
	// Извлечение ID из пути
	idStr := strings.TrimPrefix(r.URL.Path, "/api/products/")
	if idStr == "" || idStr == r.URL.Path {
		http.Error(w, "ID продукта отсутствует в пути или путь некорректен", http.StatusBadRequest)
		return
	}

	// Конвертация string в int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Некорректный формат ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProductByID(w, r, id)
	case http.MethodPut:
		h.updateProduct(w, r, id)
	case http.MethodDelete:
		h.deleteProduct(w, r, id)
	default:
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
	}
}

// createProduct обрабатывает запрос на создание нового продукта.
func (h *ProductHTTPHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Базовая валидация (более сложная валидация должна быть в usecase или отдельном слое)
	if input.Name == "" || input.Price < 0 || input.Stock < 0 {
		http.Error(w, "Имя, цена (>=0) и количество (>=0) обязательны", http.StatusBadRequest)
		return
	}
	// Валидация типа продукта
	if input.Type != models.Ticket && input.Type != models.Merchandise {
		http.Error(w, "Некорректный тип продукта. Допустимые значения: TICKET, MERCHANDISE", http.StatusBadRequest)
		return
	}

	product, err := h.productUsecase.CreateProduct(r.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			http.Error(w, "Некорректные входные данные: "+err.Error(), http.StatusBadRequest)
		} else {
			// TODO: Обработка других специфичных ошибок usecase
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

// getProductByID обрабатывает запрос на получение продукта по ID
func (h *ProductHTTPHandler) getProductByID(w http.ResponseWriter, r *http.Request, productID int) {
	product, err := h.productUsecase.GetProductByID(r.Context(), productID)
	if err != nil {
		if errors.Is(err, usecase.ErrProductNotFound) {
			http.Error(w, "Продукт не найден", http.StatusNotFound)
		} else {
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// listProducts обрабатывает запрос на получение списка всех продуктов.
func (h *ProductHTTPHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать получение параметров фильтрации и пагинации из r.URL.Query()
	// и передать их в productUsecase.ListProducts, если этот метод будет их поддерживать.

	products, err := h.productUsecase.ListProducts(r.Context())
	if err != nil {
		http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}

// updateProduct обрабатывает запрос на обновление продукта.
func (h *ProductHTTPHandler) updateProduct(w http.ResponseWriter, r *http.Request, productID int) {
	var input usecase.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Проверка, есть ли вообще что обновлять
	if input.Name == nil && input.Description == nil && input.Price == nil &&
		input.Type == nil && input.Stock == nil && input.FestivalID == nil {
		http.Error(w, "Нет данных для обновления", http.StatusBadRequest)
		return
	}
	// Дополнительная валидация для обновляемых полей
	if input.Price != nil && *input.Price < 0 {
		http.Error(w, "Цена не может быть отрицательной", http.StatusBadRequest)
		return
	}
	if input.Stock != nil && *input.Stock < 0 {
		http.Error(w, "Количество на складе не может быть отрицательным", http.StatusBadRequest)
		return
	}
	if input.Type != nil && *input.Type != models.Ticket && *input.Type != models.Merchandise {
		http.Error(w, "Некорректный тип продукта для обновления", http.StatusBadRequest)
		return
	}

	updatedProduct, err := h.productUsecase.UpdateProduct(r.Context(), productID, input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrProductNotFound):
			http.Error(w, "Продукт для обновления не найден", http.StatusNotFound)
		case errors.Is(err, usecase.ErrInvalidInput):
			http.Error(w, "Некорректные входные данные для обновления: "+err.Error(), http.StatusBadRequest)
		// TODO: Обработать другие специфичные ошибки usecase
		default:
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedProduct)
}

// deleteProduct обрабатывает запрос на удаление продукта
func (h *ProductHTTPHandler) deleteProduct(w http.ResponseWriter, r *http.Request, productID int) {
	err := h.productUsecase.DeleteProduct(r.Context(), productID)
	if err != nil {
		if errors.Is(err, usecase.ErrProductNotFound) {
			http.Error(w, "Продукт для удаления не найден", http.StatusNotFound)
		} else {
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем ответ с сообщением вместо пустого ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Продукт с ID " + strconv.Itoa(productID) + " успешно удален",
	})
}

// Получить все товары
func (h *ProductHTTPHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.ListAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// Добавить товар
func (h *ProductHTTPHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	createdProduct, err := h.repo.Create(r.Context(), &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdProduct)
}

// Удалить товар
func (h *ProductHTTPHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/products/")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// Преобразуем ID из строки в int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем ответ с сообщением вместо пустого ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Продукт с ID " + idStr + " успешно удален",
	})
}
