package grpc

import (
	"context"
	"errors"
	"strconv"

	// ВАЖНО: Замените 'your_product_module_path' на имя вашего модуля product-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	// и "github.com/Hayzerr/go-microservice-project/pb"
	pb "github.com/Hayzerr/go-microservice-project/pb" // Сгенерированные proto-файлы
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProductGRPCHandler реализует gRPC сервер для ProductService.
type ProductGRPCHandler struct {
	pb.UnimplementedProductServiceServer // Встраивание для обратной совместимости
	productUsecase                       usecase.ProductUsecase
}

// NewProductGRPCHandler создает новый экземпляр ProductGRPCHandler.
func NewProductGRPCHandler(uc usecase.ProductUsecase) *ProductGRPCHandler {
	return &ProductGRPCHandler{productUsecase: uc}
}

// mapProductModelToProto преобразует модель Product в proto-сообщение Product.
func mapProductModelToProto(product *models.Product) *pb.Product {
	if product == nil {
		return nil
	}

	// Преобразуем int в string для ID
	productID := strconv.Itoa(product.ID)

	// FestivalID может быть nil
	var festivalID string
	if product.FestivalID != nil {
		festivalID = strconv.Itoa(*product.FestivalID)
	}

	return &pb.Product{
		Id:          productID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Type:        mapProductTypeToProto(product.Type),
		Stock:       int32(product.Stock),
		FestivalId:  festivalID,
		CreatedAt:   timestamppb.New(product.CreatedAt),
		UpdatedAt:   timestamppb.New(product.UpdatedAt),
	}
}

// mapProductTypeToProto преобразует models.ProductType в pb.ProductTypeProto.
func mapProductTypeToProto(modelType models.ProductType) pb.ProductTypeProto {
	switch modelType {
	case models.Ticket:
		return pb.ProductTypeProto_TICKET
	case models.Merchandise:
		return pb.ProductTypeProto_MERCHANDISE
	default:
		return pb.ProductTypeProto_PRODUCT_TYPE_PROTO_UNSPECIFIED
	}
}

// mapProtoToProductType преобразует pb.ProductTypeProto в models.ProductType.
func mapProtoToProductType(protoType pb.ProductTypeProto) models.ProductType {
	switch protoType {
	case pb.ProductTypeProto_TICKET:
		return models.Ticket
	case pb.ProductTypeProto_MERCHANDISE:
		return models.Merchandise
	default:
		return "" // Или какое-то значение по умолчанию / ошибка
	}
}

// CreateProduct обрабатывает gRPC запрос на создание продукта.
func (h *ProductGRPCHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	// Валидация входных данных (базовая)
	if req.GetName() == "" || req.GetPrice() < 0 || req.GetStock() < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Имя, цена (>=0) и количество на складе (>=0) обязательны")
	}

	// Преобразуем festivalId из string в *int, если он задан
	var festivalID *int
	if req.GetFestivalId() != "" {
		id, err := strconv.Atoi(req.GetFestivalId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Неверный формат FestivalID: %v", err)
		}
		festivalID = &id
	}

	createInput := usecase.CreateProductInput{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Price:       req.GetPrice(),
		Type:        mapProtoToProductType(req.GetType()),
		Stock:       int(req.GetStock()), // Преобразуем int32 в int
		FestivalID:  festivalID,
	}

	product, err := h.productUsecase.CreateProduct(ctx, createInput)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			return nil, status.Errorf(codes.InvalidArgument, "Некорректные входные данные: %v", err)
		}
		// TODO: Обработка других специфичных ошибок usecase
		return nil, status.Errorf(codes.Internal, "Ошибка при создании продукта: %v", err)
	}

	return &pb.CreateProductResponse{Product: mapProductModelToProto(product)}, nil
}

// GetProduct обрабатывает gRPC запрос на получение продукта по ID.
func (h *ProductGRPCHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	productIDStr := req.GetId()
	if productIDStr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта не может быть пустым")
	}

	// Преобразуем ID из string в int
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Неверный формат ID: %v", err)
	}

	product, err := h.productUsecase.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, usecase.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "Продукт не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при получении продукта: %v", err)
	}

	return &pb.GetProductResponse{Product: mapProductModelToProto(product)}, nil
}

// ListProducts обрабатывает gRPC запрос на получение списка продуктов.
func (h *ProductGRPCHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	// TODO: Обработать параметры фильтрации и пагинации из req, если они будут добавлены

	products, err := h.productUsecase.ListProducts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении списка продуктов: %v", err)
	}

	pbProducts := make([]*pb.Product, len(products))
	for i, p := range products {
		pbProducts[i] = mapProductModelToProto(p)
	}

	return &pb.ListProductsResponse{Products: pbProducts}, nil
}

// UpdateProduct обрабатывает gRPC запрос на обновление продукта.
func (h *ProductGRPCHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	productIDStr := req.GetId()
	if productIDStr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта для обновления не может быть пустым")
	}

	// Преобразуем ID из string в int
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Неверный формат ID: %v", err)
	}

	updateInput := usecase.UpdateProductInput{}
	if req.Name != nil {
		nameVal := req.GetName().GetValue()
		updateInput.Name = &nameVal
	}
	if req.Description != nil {
		descVal := req.GetDescription().GetValue()
		updateInput.Description = &descVal
	}
	if req.Price != nil {
		priceVal := req.GetPrice().GetValue()
		updateInput.Price = &priceVal
	}
	if req.GetType() != pb.ProductTypeProto_PRODUCT_TYPE_PROTO_UNSPECIFIED {
		typeVal := mapProtoToProductType(req.GetType())
		updateInput.Type = &typeVal
	}
	if req.Stock != nil {
		stockVal := int(req.GetStock().GetValue()) // int32Value -> int
		updateInput.Stock = &stockVal
	}
	if req.FestivalId != nil {
		festIDStr := req.GetFestivalId().GetValue()
		if festIDStr != "" {
			festIDVal, err := strconv.Atoi(festIDStr)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Неверный формат FestivalID: %v", err)
			}
			updateInput.FestivalID = &festIDVal
		}
	}

	if updateInput.Name == nil && updateInput.Description == nil && updateInput.Price == nil &&
		updateInput.Type == nil && updateInput.Stock == nil && updateInput.FestivalID == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Нет данных для обновления")
	}

	updatedProduct, err := h.productUsecase.UpdateProduct(ctx, productID, updateInput)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrProductNotFound):
			return nil, status.Errorf(codes.NotFound, "Продукт для обновления не найден: %v", err)
		case errors.Is(err, usecase.ErrInvalidInput):
			return nil, status.Errorf(codes.InvalidArgument, "Некорректные входные данные для обновления: %v", err)
		// TODO: Обработать другие специфичные ошибки usecase
		default:
			return nil, status.Errorf(codes.Internal, "Ошибка при обновлении продукта: %v", err)
		}
	}
	return &pb.UpdateProductResponse{Product: mapProductModelToProto(updatedProduct)}, nil
}

// DeleteProduct обрабатывает gRPC запрос на удаление продукта.
func (h *ProductGRPCHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*emptypb.Empty, error) {
	productIDStr := req.GetId()
	if productIDStr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта для удаления не может быть пустым")
	}

	// Преобразуем ID из string в int
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Неверный формат ID: %v", err)
	}

	err = h.productUsecase.DeleteProduct(ctx, productID)
	if err != nil {
		if errors.Is(err, usecase.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "Продукт для удаления не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при удалении продукта: %v", err)
	}

	return &emptypb.Empty{}, nil
}
