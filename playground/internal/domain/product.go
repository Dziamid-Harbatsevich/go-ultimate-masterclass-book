package domain

import (
	"errors"
	"fmt"
)

// Product represents the core domain model asset
type Product struct {
	ID    int
	Name  string
	Price float64
}

// ProductStore defines the implicit data-access layer contract
type ProductStore interface {
	Save(p *Product) error
	FindByID(id int) (*Product, error)
}

// ProductService manages core business operations and embeds the interface contract dependency
type ProductService struct {
	Store ProductStore // Decoupled dependency injection via interface type
}

// RegisterNewProduct enforces validation rules before persisting data
func (s *ProductService) RegisterNewProduct(name string, price float64) (*Product, error) {
	if name == "" {
		return nil, errors.New("domain validation error: product name is required")
	}
	if price <= 0 {
		return nil, errors.New("domain validation error: product price must be greater than zero")
	}

	newProd := &Product{
		Name:  name,
		Price: price,
	}

	// Persist data using the decoupled storage interface contract
	if err := s.Store.Save(newProd); err != nil {
		return nil, fmt.Errorf("domain persistence failure: %w", err)
	}

	return newProd, nil
}
