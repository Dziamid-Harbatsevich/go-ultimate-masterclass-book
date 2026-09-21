package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	// Import custom workspace files using your module name prefix
	"playground/internal/config"
	"playground/internal/data"
	"playground/internal/db"
	"playground/internal/domain"
	internalConfig "playground/pkg/config"

	_ "github.com/go-sql-driver/mysql"
)

// MockProductDB implements domain.ProductStore implicitly by defining matching methods
type MockProductDB struct {
	memoryTable map[int]*domain.Product
	currentID   int
}

type Env struct{}

type ApplicationContainer struct {
	Products *data.ProductRepository
}

type ErrorResponse struct {
	Message string `json:"error"`
}

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

func (app *ApplicationContainer) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Invalid input token identifier"})
		return
	}

	// Establish a bounded 3-second processing timeout context for this database pipeline operation
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	product, err := app.Products.FindByID(ctx, id)
	if err != nil {
		if err.Error() == "product record not found" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Message: err.Error()})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Internal operational data lookup failure"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}

func (app *ApplicationContainer) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Read and decode the inbound JSON stream directly from the network socket reader
	var input struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Malformed input JSON structure"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := app.Products.CreateWithAudit(ctx, input.Name, input.Price)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Transaction processing failure encountered"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
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

	//-- DAY 3
	fmt.Println("=== DAY 3: MULTI-PACKAGE MICROSERVICE RUNNER ===")

	// 1. Load configurations using public helper elements inside /pkg
	inputDSN := "app_admin:secure_mysql_pass@tcp(127.0.0.1:3306)/production_store"
	targetPort := "8080"

	profile, err := internalConfig.LoadProfile(inputDSN, targetPort)
	if err != nil {
		fmt.Printf("Initialization Fatal Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Public configuration assets loaded cleanly.")
	fmt.Printf("[Config] Networking targeted port listener mapped to %s\n", profile.APIPort)

	// 2. Initialize connection structures using systems isolated inside /internal
	connectionPool, err := db.BootPool(profile.DSN)
	if err != nil {
		fmt.Printf("Data Initialization Fatal Error: %v\n", err)
		os.Exit(1)
	}

	if connectionPool.IsConnected {
		fmt.Println("🚀 System booted successfully. Microservice is online.")
	}

	//-- DAY 4
	// app := &Env{}
	// mux := http.NewServeMux()

	// mux.HandleFunc("POST /products", app.CreateProductEndpoint)

	// server := &http.Server{
	// 	Addr:         ":8080",
	// 	Handler:      mux,
	// 	ReadTimeout:  5 * time.Second,
	// 	WriteTimeout: 10 * time.Second,
	// }

	// log.Println("🚀 Day 4 Validation & Error Core Service actively listening on port 8080...")
	// log.Fatal(server.ListenAndServe())

	//-- DAY 5
	fmt.Println("Starting Day 5. MySQL ACID Transactions...")

	// Initialize a production connection pool (Update with your local credentials)
	dsn := "test:password@tcp(127.0.0.1:3306)/test?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Pool activation crash: %v", err)
	}
	defer db.Close()

	// Tune connection limits to match production capacity profiles
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection check failed: %v", err)
	}
	log.Println("🔌 Core MySQL connection pool initialized successfully.")

	repo := &data.ProductRepository{DB: db}
	appWithRealSql := &ApplicationContainer{Products: repo}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", appWithRealSql.GetProductHandler)
	mux.HandleFunc("POST /products", appWithRealSql.CreateProductHandler)

	log.Println("🚀 High-Performance REST Engine listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
