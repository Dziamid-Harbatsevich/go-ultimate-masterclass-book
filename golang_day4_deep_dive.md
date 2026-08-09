# 📘 Day 4: Advanced Error Architecture, Sentinel Errors, and REST API Payload Validation

## Course Executive Summary

* **The Return Value Paradigm Shift**: Transitions senior developers from implicit runtime exception capturing (`try/catch`) to explicit compile-time control flows. Go treats errors as first-class, inspectable values returned explicitly by functions.
* **Error Wrapping and Unwrapping**: Deep-dive mechanics of leveraging `%w` verbs with `fmt.Errorf` to preserve contextual execution stacks without destroying underlying causal roots.
* **Type-Safe Inspection (`errors.Is` vs. `errors.As`)**: Definitive diagnostic strategies to safely extract sentinel values or custom error structs out of a nested structural chain.
* **MySQL Database Error Translation**: Concrete structural patterns to isolate database anomalies (duplicate keys, deadlocks, missing records) inside the persistence layer and map them to explicit domain-driven behaviors.
* **REST API Input Data Validation**: Building a lightweight, highly performant validation engine using custom maps to accumulate schema processing errors into structured JSON arrays without relying on massive, magic-heavy reflection libraries.
* **Enterprise Structural Lab**: A complete, multi-package production-ready implementation (`errors.go`, `validator.go`, `products.go`, `handlers.go`, `main.go`) validating inbound REST payloads and mapping database states cleanly.

---

## 1. The Architectural Paradigm Shift: Errors as First-Class Values

For engineers coming from PHP (Exception throwing), C# (`throw new Exception`), or TypeScript, Go’s error handling model can initially feel verbose. However, it represents a monumental leap forward for backend predictability, stability, and runtime optimization.

### The Problem with Implicit Exceptions
In languages with traditional exceptions, an execution thread can bubble up through dozens of layers implicitly. If an unhandled exception hits the global router, it often results in a generic `500 Internal Server Error`, leaks underlying stack traces, or silently leaves database connections open. You cannot easily see all code failure states by reading a method's signature.

### The Go Design Philosophy
Go does not have exceptions that break the normal execution flow. Instead, Go treats errors as explicit values. The standard library defines the built-in `error` type as a simple interface:

```go
type error interface {
    Error() string
}
```

Any structure that implements a single `Error() string` method is a valid error. Because functions return multiple values, the last return argument is traditionally reserved for this `error`. 

This turns error tracing into an explicit, compile-time control flow challenge. You handle the error immediately where it occurs, ensuring absolute predictability over your code paths.

---

## 2. Sentinel Errors vs. Custom Error Structs

To build microservices, you need to differentiate between two core classification strategies: **Sentinel Errors** and **Custom Error Structs**.

### A. Sentinel Errors (State Constants)
Sentinel errors are fixed variables used to signify a specific state or boundary condition. The term comes from old programming paradigms where a specific "sentinel value" indicated that a process was complete.
* **When to use**: When your calling code only needs to know *what* happened, without needing extra contextual metadata or dynamic values.
* **Syntax**: Created using the `errors.New()` function or `fmt.Errorf()`.

```go
package data

import "errors"

// Idiomatic Naming Convention: Prefix with "Err"
var (
    ErrRecordNotFound = errors.New("data: resource record not found")
    ErrDuplicateEmail = errors.New("data: email address already exists")
    ErrDatabaseClosed = errors.New("data: network connection pool closed")
)
```

### B. Custom Error Structs (Contextual Payload Objects)
When an error requires dynamic metadata (such as tracking which specific database field triggered a validation failure, or capturing a timestamp/tracking ID), you define a custom struct that implements the `error` interface.
* **When to use**: When you need to pass additional rich data alongside the failure message up to the REST handler layer.

```go
package data

import "fmt"

type QueryExecutionError struct {
	Query     string
	Underlying error
	Timestamp string
}

// Satisfy the error interface implicitly
func (e *QueryExecutionError) Error() string {
	return fmt.Sprintf("database crash at %s during query [%s]: %v", e.Timestamp, e.Query, e.Underlying)
}
```

---

## 3. Error Wrapping, Unwrapping, and Inspection Mechanics

As an error travels up from your database persistence repository to your business logic, and finally to your HTTP controller handlers, it often needs to accumulate contextual details without obscuring the original underlying cause.

### Modern Error Wrapping (`%w`)
Go introduces the `%w` formatting verb inside `fmt.Errorf`. This creates a nested chain of errors, preserving the original error value like an onion shell.

```go
// Inside internal/data/products.go
if err := db.Exec(query); err != nil {
    // Wrapping the raw driver error with our custom domain context
    return fmt.Errorf("failed to register new user profile: %w", err)
}
```

### Advanced Type-Safe Inspection
Never use string matching (like `strings.Contains(err.Error(), "not found")`) to identify error types in production. It is fragile and breaks easily if text formatting changes. Instead, use the type-safe inspection tools provided by the `errors` package:

#### 1. `errors.Is` (Checking for Sentinels)
Compares an error against a specific target value. It automatically traverses down the entire wrapped error chain to look for a match.

```go
if errors.Is(err, data.ErrRecordNotFound) {
    // Map directly to an HTTP 404 Not Found status
}
```

#### 2. `errors.As` (Extracting Custom Error Payload Structures)
Checks whether an error matches a specific custom error structure type. If a match is found, it casts the error directly into that target structure pointer variable, giving you immediate access to its custom fields.

```go
var validationErr *data.CustomValidationError
if errors.As(err, &validationErr) {
    // You now have type-safe access to validationErr.FailedFields
}
```

---

## 4. Mapping MySQL Driver Errors into Domain Behaviors

To maintain clean architectural boundaries, your outer HTTP controller layers should never import or read driver-specific databases structures directly (like `github.com/go-sql-driver/mysql`). 

Instead, your repository layer must catch these driver-specific anomalies, extract their low-level codes, and map them to clean, reusable domain sentinel values.

### 💻 Code Blueprint: Catching MySQL Error Codes Safely

```go
package data

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
)

var (
	ErrDuplicateEntry = errors.New("resource conflict: record already exists")
	ErrPersistenceFail = errors.New("internal storage system failure")
)

type ProductModel struct {
	DB *sql.DB
}

func (m *ProductModel) SaveProduct(name string) error {
	query := "INSERT INTO products (name) VALUES (?)"
	_, err := m.DB.Exec(query, name)
	
	if err != nil {
		// 1. Declare a pointer to the specific MySQL driver error type
		var mysqlErr *mysql.MySQLError
		
		// 2. Safely extract the low-level MySQL driver code using errors.As
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062: // MySQL Duplicate Entry Error Code
				return ErrDuplicateEntry
			case 1213: // MySQL Deadlock Detected Error Code
				return errors.New("database deadlock encountered, transaction aborted")
			}
		}
		
		return ErrPersistenceFail
	}
	return nil
}
```

---

## 5. Building an Explicit REST API Data Validation Engine

In frameworks like Laravel (PHP) or NestJS (TypeScript), validation often relies on complex annotations, attributes, or magic reflection rules. Go values clarity and performance. You can build an incredibly fast, type-safe validation engine using standard library maps.

```
       REST INPUT PAYLOAD VALIDATION PIPELINE
┌────────────────────────┐
│  Inbound JSON Payload  │ ──► {"name": "", "price": -5.00}
└───────────┬────────────┘
            │ (json.NewDecoder)
            ▼
┌────────────────────────┐
│  Native Bound Struct   │ ──► Struct Field Tag Mapping
└───────────┬────────────┘
            │
            ▼
┌────────────────────────┐
│    Validator Engine    │ ──► Map Validation Evaluation Rules
└───────────┬────────────┘
            │
            ├── Valid? (Yes) ──► Pass into MySQL Repository Storage
            │
            └── Valid? (No)  ──► Halt Execution & Return HTTP 422 JSON Map
```

---

## 6. Real-World Deep Dive Lab: Production-Grade REST Error & Validation Engine

Let's build a fully functioning, thread-safe, production-ready validation and error translation layer. This implementation follows strict microservice conventions and handles schema parsing and error formatting without using heavy third-party framework wrappers.

### 📁 Multi-Package Architecture Directory Map
Ensure your local project workspace is organized using these precise file blocks:
```text
go-mysql-api/
├── cmd/
│   └── api/
│       ├── handlers.go
│       ├── errors.go
│       └── main.go
└── internal/
    └── validator/
        └── validator.go
```

### 📜 File: `internal/validator/validator.go`
Create the core validation utility package. It uses a clean map structure to accumulate parsing errors.

```go
package validator

// Validator serves as the centralized structure for tracking payload errors
type Validator struct {
	Errors map[string]string
}

// New instantiates a fresh validation container
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// HasErrors returns true if any issues are currently logged
func (v *Validator) HasErrors() bool {
	return len(v.Errors) > 0
}

// AddError logs a distinct validation error message if the field isn't already tracked
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the log only if an evaluation rule evaluates to false
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}
```

### 📜 File: `cmd/api/errors.go`
Create your centralized HTTP error utility functions to guarantee clean, consistent REST API JSON outputs across all application states.

```go
package main

import (
	"log"
	"net/http"
	"encoding/json"
)

// JSONErrorResponse creates a uniform error contract matching standard microservice criteria
type JSONErrorResponse struct {
	Status int               `json:"status"`
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// writeErrorPayload streams formatted error data down the active network pipe
func writeErrorPayload(w http.ResponseWriter, status int, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	payload := JSONErrorResponse{
		Status: status,
		Error:  message,
		Fields: fields,
	}
	
	json.NewEncoder(w).Encode(payload)
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeErrorPayload(w, status, message, nil)
}

func serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("❌ SERVER DESCRIPTOR FAILURE: %s %s | error: %v", r.Method, r.URL.Path, err)
	message := "The server encountered an internal processing breakdown"
	errorResponse(w, r, http.StatusInternalServerError, message)
}

func badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	writeErrorPayload(w, http.StatusUnprocessableEntity, "schema validation rules failed", errors)
}
```

### 📜 File: `cmd/api/handlers.go`
Create your primary REST controllers. These process validation checks and intercept wrapped error values cleanly.

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-mysql-api/internal/validator"
	"net/http"
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
```

### 📜 File: `cmd/api/main.go`
Create the operational bootstrap file to stitch your application logic paths together:

```go
package main

import (
	"log"
	"net/http"
	"time"
)

type Env struct{}

func main() {
	app := &Env{}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /products", app.CreateProductEndpoint)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("🚀 Day 4 Validation & Error Core Service actively listening on port 8080...")
	log.Fatal(server.ListenAndServe())
}
```

---

## 🏃‍♂️ Verification Protocols

Launch your service inside your local terminal window layout:
```bash
go run ./cmd/api/*.go
```

Open an independent diagnostic terminal tab to test your error and validation handling behaviors:

### Test Case A: Triggering Validation Constraints (HTTP 422)
```bash
curl -i -X POST http://localhost:8080/products   -H "Content-Type: application/json"   -d '{"sku": "", "name": "", "price": -10.00}'
```

### Test Case B: Triggering Domain Conflict Sentinel Verification (HTTP 409)
```bash
curl -i -X POST http://localhost:8080/products   -H "Content-Type: application/json"   -d '{"sku": "SKU-DUPE", "name": "Premium Keyboard", "price": 99.00}'
```

### Test Case C: Triggering Internal Server Fallbacks (HTTP 500)
```bash
curl -i -X POST http://localhost:8080/products   -H "Content-Type: application/json"   -d '{"sku": "SKU-CRASH", "name": "Broken Item", "price": 5.00}'
```
