package grpc

import (
	"context"
	"errors"

	// ВАЖНО: Замените 'your_product_module_path' на имя вашего модуля product-service из go.mod
	// Например: "github.com/Hayzerr/go-microservice-project/product-service/internal/product/models"
	// и "github.com/Hayzerr/go-microservice-project/pb"
	"your_product_module_path/internal/product/models"
	"your_product_module_path/internal/product/usecase"
	pb "your_product_module_path/pb" // Сгенерированные proto-файлы

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ProductGRPCHandler реализует gRPC сервер для ProductService.
type ProductGRPCHandler struct {
	pb.UnimplementedProductServiceServer // Встраивание для обратной совместимости
	productUsecase                 usecase.ProductUsecase
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
	return &pb.Product{
		Id:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Type:        mapProductTypeToProto(product.Type), // Используем маппер для enum
		Stock:       int32(product.Stock),                // Преобразуем int в int32
		FestivalId:  product.FestivalID,
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

	createInput := usecase.CreateProductInput{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Price:       req.GetPrice(),
		Type:        mapProtoToProductType(req.GetType()),
		Stock:       int(req.GetStock()), // Преобразуем int32 в int
		FestivalID:  req.GetFestivalId(),
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
	productID := req.GetId()
	if productID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта не может быть пустым")
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
	productID := req.GetId()
	if productID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта для обновления не может быть пустым")
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
	// Для enum ProductTypeProto, если он не обернут в wrappers.Value,
	// он всегда будет иметь значение (даже если это *_UNSPECIFIED).
	// Если вы хотите сделать его опциональным, его тоже нужно обернуть или
	// добавить специальное значение "не изменять".
	// В данном случае, если передано PRODUCT_TYPE_PROTO_UNSPECIFIED, мы можем это игнорировать.
	if req.GetType() != pb.ProductTypeProto_PRODUCT_TYPE_PROTO_UNSPECIFIED {
		typeVal := mapProtoToProductType(req.GetType())
		updateInput.Type = &typeVal
	}
	if req.Stock != nil {
		stockVal := int(req.GetStock().GetValue()) // int32Value -> int
		updateInput.Stock = &stockVal
	}
	if req.FestivalId != nil {
		festIDVal := req.GetFestivalId().GetValue()
		updateInput.FestivalID = &festIDVal
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
	productID := req.GetId()
	if productID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID продукта для удаления не может быть пустым")
	}

	err := h.productUsecase.DeleteProduct(ctx, productID)
	if err != nil {
		if errors.Is(err, usecase.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "Продукт для удаления не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при удалении продукта: %v", err)
	}

	return &emptypb.Empty{}, nil
}
