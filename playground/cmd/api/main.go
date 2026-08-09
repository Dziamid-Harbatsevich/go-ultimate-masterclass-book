package main

import (
	"bootstrapper/internal/config"
	"bootstrapper/internal/domain"
	"errors"
	"fmt"
	"os"
)

func bootstrapDB() {
	// 1. Instantiate the memory object space inside the application Stack
	var appConfig config.Environment

	// 2. Mock environment variables (In production, these come from os.Getenv)
	rawInputDSN := "root:root_pass@tcp(127.0.0.1:3306)/app_prod_db"
	allocatedWorkers := 50

	// 3. Pass a memory pointer to the configuration parser
	err := config.ParseCredentials(&appConfig, rawInputDSN, allocatedWorkers)

	// 4. Verify the initialization process explicitly
	if err != nil {
		fmt.Printf("Critical System Initialization Failure: %v\n", err)
		os.Exit(1)
	}

	// 5. Output configuration success parameters using the public structural fields
	fmt.Println("Configuration verification complete.")
	fmt.Printf("Database Endpoint Target: %s\n", appConfig.DSN)
	fmt.Printf("Total Thread Workers Configured: %d\n", appConfig.MaxWorkers)
	fmt.Println("🚀 Application booted cleanly without warnings.")
}

// MockProductDB implements domain.ProductStore implicitly by defining matching methods
type MockProductDB struct {
	memoryTable map[int]*domain.Product
	currentID   int
}

// Save stores the product into an in-memory hash map table
func (db *MockProductDB) Save(p *domain.Product) error {
	db.currentID++
	p.ID = db.currentID
	db.memoryTable[p.ID] = p
	return nil
}

// FindByID retrieves a product record from memory or returns an error
func (db *MockProductDB) FindByID(id int) (*domain.Product, error) {
	p, exists := db.memoryTable[id]
	if !exists {
		return nil, errors.New("sql: no rows found in result set")
	}
	return p, nil
}

func main() {

	fmt.Println("Initializing Day 1 microservice architecture...")
	bootstrapDB()

	fmt.Println("Starting Day 2 decoupling and polymorphism verification loop...")

	// 1. Initialize the mock memory storage pool
	mockStore := &MockProductDB{
		memoryTable: make(map[int]*domain.Product),
	}

	// 2. Instantiate your core business service layer by injecting the mock store
	service := &domain.ProductService{
		Store: mockStore, // Valid because *MockProductDB implements domain.ProductStore
	}

	// 3. Register a valid product via the service layer
	product, err := service.RegisterNewProduct("Ultrawide Curved Monitor", 649.99)
	if err != nil {
		fmt.Printf("Execution Error: %v", err)
		os.Exit(1)
	}

	fmt.Println("🚀 Registration processing execution succeeded!")
	fmt.Printf("Assigned Row ID: %d\n", product.ID)
	fmt.Printf("Stored Model Name: %s\n", product.Name)
	fmt.Printf("Stored Model Price: %.2f\n", product.Price)

	// 4. Test service validation rules with invalid inputs
	_, err = service.RegisterNewProduct("", -10.00)
	if err != nil {
		fmt.Printf("\nValidation intercept verification passed: %v\n", err)
	}
}
