package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"playground/internal/validator"
)

// Simulated Domain Sentinel Errors for the sake of the lab
var (
	ErrDuplicateSKU  = errors.New("domain conflict: product SKU code already exists")
	ErrStorageSystem = errors.New("domain persistence system layer failed")
)

type ProductPayload struct {
	SKU   string  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// MockDatabaseSaver simulates our persistence layer and demonstrates error wrapping
func MockDatabaseSaver(sku string) error {
	if sku == "SKU-DUPE" {
		// Wrap the original domain error to show how context is preserved
		return fmt.Errorf("repository action rejected: %w", ErrDuplicateSKU)
	}
	if sku == "SKU-CRASH" {
		return ErrStorageSystem
	}
	return nil
}

func (app *Env) CreateProductEndpoint(w http.ResponseWriter, r *http.Request) {
	var input ProductPayload

	// 1. Decode incoming JSON payload stream safely
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		badRequestResponse(w, r, errors.New("malformed input structural payload body"))
		return
	}

	// 2. Initialize the type-safe validation engine
	v := validator.New()

	// 3. Enforce strict, explicit business rule checks
	v.Check(input.SKU != "", "sku", "product stock keeping unit code is required")
	v.Check(input.Name != "", "name", "product display title cannot be empty")
	v.Check(input.Price > 0, "price", "product price parameters must be positive numbers")

	// 4. Halt execution early if validation checks fail
	if v.HasErrors() {
		failedValidationResponse(w, r, v.Errors)
		return
	}

	// 5. Execute storage operations and intercept errors explicitly
	err := MockDatabaseSaver(input.SKU)
	if err != nil {
		// Inspect error chains safely using type-safe errors.Is checks
		if errors.Is(err, ErrDuplicateSKU) {
			errorResponse(w, r, http.StatusConflict, "A product item with that identical SKU code is already registered")
			return
		}

		// Fallback for unhandled internal failures
		serverErrorResponse(w, r, err)
		return
	}

	// 6. Return successful response payload on clean execution paths
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Product successfully registered"})
}
