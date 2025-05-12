package usecase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	// ВАЖНО: Замените 'your_product_module_path' на имя вашего модуля product-service из go.mod
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/repository"
)

var (
	ErrProductNotFound = errors.New("продукт не найден")
	ErrInvalidInput    = errors.New("некорректные входные данные")
	// Добавьте другие ошибки бизнес-логики, если необходимо
)

// CreateProductInput определяет структуру для входных данных при создании продукта.
type CreateProductInput struct {
	Name        string
	Description string
	Price       float64
	Type        models.ProductType
	Stock       int
	FestivalID  string
}

// UpdateProductInput определяет структуру для входных данных при обновлении продукта.
type UpdateProductInput struct {
	Name        *string
	Description *string
	Price       *float64
	Type        *models.ProductType
	Stock       *int
	FestivalID  *string
}

// ProductUsecase определяет интерфейс для бизнес-логики, связанной с продуктами.
type ProductUsecase interface {
	CreateProduct(ctx context.Context, input CreateProductInput) (*models.Product, error)
	GetProductByID(ctx context.Context, id string) (*models.Product, error)
	ListProducts(ctx context.Context) ([]*models.Product, error)
	UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*models.Product, error)
	DeleteProduct(ctx context.Context, id string) error
}

type productUsecase struct {
	productRepo repository.ProductRepository
	// Здесь могут быть другие зависимости, например, клиент к сервису фестивалей
}

// NewProductUsecase создает новый экземпляр productUsecase.
func NewProductUsecase(productRepo repository.ProductRepository) ProductUsecase {
	return &productUsecase{
		productRepo: productRepo,
	}
}

// CreateProduct создает новый продукт.
func (uc *productUsecase) CreateProduct(ctx context.Context, input CreateProductInput) (*models.Product, error) {
	// Валидация входных данных
	if input.Name == "" || input.Price < 0 || input.Stock < 0 {
		return nil, ErrInvalidInput
	}
	// Дополнительная валидация типа продукта, festival_id и т.д.

	product := &models.Product{
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Type:        input.Type,
		Stock:       input.Stock,
		FestivalID:  input.FestivalID,
	}

	createdProduct, err := uc.productRepo.Create(ctx, product)
	if err != nil {
		// TODO: Обработка специфичных ошибок репозитория
		return nil, err
	}
	return createdProduct, nil
}

// GetProductByID находит продукт по ID.
func (uc *productUsecase) GetProductByID(ctx context.Context, id string) (*models.Product, error) {
	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err // Ошибка репозитория
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

// ListProducts возвращает список всех продуктов.
func (uc *productUsecase) ListProducts(ctx context.Context) ([]*models.Product, error) {
	products, err := uc.productRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	// Если список пуст, возвращаем пустой слайс, а не ошибку
	if products == nil {
		return []*models.Product{}, nil
	}
	return products, nil
}

// UpdateProduct обновляет существующий продукт.
func (uc *productUsecase) UpdateProduct(ctx context.Context, id string, input UpdateProductInput) (*models.Product, error) {
	currentProduct, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if currentProduct == nil {
		return nil, ErrProductNotFound
	}

	productToUpdate := *currentProduct
	changed := false

	// Обновляем поля, если они предоставлены
	if input.Name != nil && *input.Name != productToUpdate.Name {
		productToUpdate.Name = *input.Name
		changed = true
	}
	if input.Description != nil && *input.Description != productToUpdate.Description {
		productToUpdate.Description = *input.Description
		changed = true
	}
	if input.Price != nil && *input.Price != productToUpdate.Price {
		if *input.Price < 0 {
			return nil, ErrInvalidInput
		} // Валидация цены
		productToUpdate.Price = *input.Price
		changed = true
	}
	if input.Type != nil && *input.Type != productToUpdate.Type {
		// TODO: Валидация нового типа, если необходимо
		productToUpdate.Type = *input.Type
		changed = true
	}
	if input.Stock != nil && *input.Stock != productToUpdate.Stock {
		if *input.Stock < 0 {
			return nil, ErrInvalidInput
		} // Валидация остатка
		productToUpdate.Stock = *input.Stock
		changed = true
	}
	if input.FestivalID != nil && *input.FestivalID != productToUpdate.FestivalID {
		// TODO: Валидация festival_id, если необходимо
		productToUpdate.FestivalID = *input.FestivalID
		changed = true
	}

	if !changed {
		return currentProduct, nil // Ничего не изменилось
	}

	updatedProduct, err := uc.productRepo.Update(ctx, &productToUpdate)
	if err != nil {
		// TODO: Обработка специфичных ошибок репозитория
		return nil, err
	}
	return updatedProduct, nil
}

// DeleteProduct удаляет продукт по ID.
func (uc *productUsecase) DeleteProduct(ctx context.Context, id string) error {
	err := uc.productRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound // Преобразуем ошибку репозитория
		}
		return err
	}
	return nil
}
