package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/leatherstore/catalog-service/internal/domain"
	"github.com/leatherstore/catalog-service/internal/repository"
)

const defaultListLimit = 1000

type DefaultProductService struct {
	products repository.ProductRepository
}

func NewProductService(products repository.ProductRepository) ProductService {
	return &DefaultProductService{products: products}
}

func (s *DefaultProductService) CreateProduct(ctx context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
	}
	if err := s.products.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *DefaultProductService) UpdateProduct(ctx context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
	}
	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *DefaultProductService) GetProductByID(ctx context.Context, id int64) (*domain.LeatherProduct, error) {
	return s.products.FindByID(ctx, id)
}

func (s *DefaultProductService) GetProductBySKU(ctx context.Context, sku string) (*domain.LeatherProduct, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, fmt.Errorf("%w: SKU is required", ErrInvalidProduct)
	}
	return s.products.FindBySKU(ctx, sku)
}

func (s *DefaultProductService) ListProducts(ctx context.Context) ([]domain.LeatherProduct, error) {
	return s.products.List(ctx, defaultListLimit, 0)
}

func (s *DefaultProductService) DeleteProduct(ctx context.Context, id int64) error {
	return s.products.Delete(ctx, id)
}

func (s *DefaultProductService) ActivateProduct(ctx context.Context, id int64) error {
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return err
	}
	p.Active = true
	return s.products.Update(ctx, p)
}

func (s *DefaultProductService) DeactivateProduct(ctx context.Context, id int64) error {
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return err
	}
	p.Active = false
	return s.products.Update(ctx, p)
}

func (s *DefaultProductService) UpdateInventory(ctx context.Context, sku string, quantity int, stockStatus domain.StockStatus) (*domain.LeatherProduct, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, fmt.Errorf("%w: SKU is required", ErrInvalidProduct)
	}
	if quantity < 0 {
		return nil, fmt.Errorf("%w: stock quantity must not be negative", ErrInvalidProduct)
	}
	if !stockStatus.IsValid() {
		return nil, fmt.Errorf("%w: invalid stock status", ErrInvalidProduct)
	}

	p, err := s.products.FindBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	p.StockQuantity = quantity
	p.StockStatus = stockStatus

	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *DefaultProductService) UpdatePricing(ctx context.Context, sku string, retailPrice domain.Money, salePrice *domain.Money, currency string) (*domain.LeatherProduct, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, fmt.Errorf("%w: SKU is required", ErrInvalidProduct)
	}
	if retailPrice.AmountCents < 0 {
		return nil, fmt.Errorf("%w: retail price must not be negative", ErrInvalidProduct)
	}
	if salePrice != nil && salePrice.AmountCents < 0 {
		return nil, fmt.Errorf("%w: sale price must not be negative", ErrInvalidProduct)
	}
	if strings.TrimSpace(currency) == "" {
		return nil, fmt.Errorf("%w: currency is required", ErrInvalidProduct)
	}

	p, err := s.products.FindBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	p.RetailPrice = retailPrice
	p.SalePrice = salePrice
	p.Currency = currency

	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
