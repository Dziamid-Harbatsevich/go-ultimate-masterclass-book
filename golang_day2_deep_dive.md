# 📘 Day 2: Composition Over Inheritance & Polymorphism Semantics

## 📝 Printed Summary for Day 2

- **Composition Over Inheritance Mechanics**: Deep operational explanation of how Go bypasses the "Fragile Base Class" architectural vulnerabilities of classical OOP (PHP/C#) by nesting structs anonymously instead of deriving child dependencies.
- **Property Promotion & Shadowing Collision Resolution**: Complete structural logic maps showing how properties bubble up to outer structs automatically, alongside strategies for resolving naming collisions explicitly using typed namespaces.
- **Value vs. Pointer Receivers Performance Balancing**: Technical criteria evaluating when to execute actions on a local stack clone (Value `(u User)`) versus changing original data points directly across global memory regions via pointer addresses (`(u *User)`).
- **Decoupled Interface Polymorphism (Implicit Duck Typing)**: Architectural rules for designing production layers where structs automatically satisfy domain contracts simply by implementing matching methods, removing the need for boilerplate compilation keywords like `implements`.
- **The Domain Isolation Lab**: A complete, multi-file code implementation (`product.go`, `main.go`) utilizing a fast, memory-isolated mock data engine to test business constraints completely decoupled from standard persistence drives.

---

## 1. The Architectural Paradigm Shift: Composition vs. Inheritance

For engineers with an intensive background in classical Object-Oriented Programming (OOP) like PHP (Symfony/Laravel), C#, or TypeScript, Go requires a complete shift in how you structure data and behaviors.

Go intentionally lacks a `class` keyword, an `extends` keyword, and traditional hierarchical inheritance. Instead, it relies strictly on **Composition Over Inheritance** using **Struct Embedding** and **Implicit Interfaces**.

### The Fragile Base Class Problem (Classical OOP)

In classical architectures, code reuse often leads to deep hierarchical trees:

```
BaseModel ──► AuthenticatableModel ──► User ──► AdminUser
```

This pattern introduces tight coupling. A modification to the internal behavior of `BaseModel` can inadvertently cause breaking changes down the entire chain of child structures.

### The Go Composition Model (Has-A / Acts-As)

Go structures are decoupled. You build complex models by combining small, focused components.

- A struct doesn't *inherit* from another; it **embeds** it (Composition).
- A struct doesn't explicitly declare that it *implements* a contract; it satisfies it automatically by defining the matching methods (Implicit Interface).

---

## 2. Struct Embedding, Anonymous Fields, and Promotion Mechanics

Go allows you to embed one struct directly inside another without giving it an explicit field name. This is called **Anonymous Struct Embedding**. When you do this, the fields and methods of the embedded struct are automatically **promoted** to the outer struct.

### 💻 Code Blueprint: Field Promotion and Shadowing Rules

```go
package main

import "fmt"

// AuditModel represents metadata shared across database rows
type AuditModel struct {
	ID        int
	CreatedAt string
}

// User composes AuditModel anonymously to acquire its properties
type User struct {
	AuditModel // Anonymous field embedding
	Email      string
	ID         int // Field Shadowing: This overrides AuditModel.ID on the top-level
}

func main() {
	// Instantiating a composed structure requires nested initialization
	u := User{
		AuditModel: AuditModel{
			ID:        1001,
			CreatedAt: "2026-07-25",
		},
		Email: "senior_dev@domain.com",
		ID:    42,
	}

	// 1. Direct Promotion: CreatedAt is accessible directly from the outer struct
	fmt.Println("Promoted Field:", u.CreatedAt) // Output: 2026-07-25

	// 2. Field Shadowing: Accessing ID gives the outer struct's value
	fmt.Println("Outer Shadowed ID:", u.ID) // Output: 42

	// 3. Explicit Access: The inner struct's fields remain accessible via its type name
	fmt.Println("Inner Hidden ID:", u.AuditModel.ID) // Output: 1001
}
```

---

## 3. Receiver Semantics: Value vs. Pointer Receivers

In Go, you attach functions to structs using **method receivers**. This is the closest equivalent to defining a method inside a class. There are two types of receivers: **Value Receivers** and **Pointer Receivers**. Choosing the right one determines your application's performance, thread safety, and mutation behavior.

```
METHOD RECEIVER COMPARISON PROFILE
┌─────────────────────────────────────────────────────────────────────────────┐
│ 👥 VALUE RECEIVER: func (u User) SendEmail()                                │
│ * Copies the entire struct on the Stack.                                    │
│ * Original struct is completely immutable; modifications remain local.       │
│ * Thread-safe for concurrent read-only actions across multiple goroutines.  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
┌─────────────────────────────────────────────────────────────────────────────┐
│ 📍 POINTER RECEIVER: func (u *User) UpdatePassword(p string)                │
│ * Passes a 64-bit memory address directly.                                  │
│ * Mutates the original struct fields directly in memory.                    │
│ * High-performance; eliminates memory copying overhead for large objects.   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Senior Execution Rules for Microservice Gateways

- **Consistency Rule**: If any method on a struct requires a **pointer receiver** to mutate data, *all* methods on that struct should use a pointer receiver, even if they only read data. This ensures consistent interface resolution at compile time.
- **Size Threshold**: If a struct contains strings, arrays, or maps that exceed 64 bytes, use pointer receivers across the board to avoid the CPU cost of copying data on the stack during rapid execution loops.

---

## 4. Polymorphism and Implicit Interfaces (Duck Typing)

Go interfaces are completely implicit. Unlike PHP (`interface UserRepo { ... }` and `class MySQLRepo implements UserRepo`), a Go struct does not declare its interface compliance. If a struct defines the exact methods listed in an interface, the compiler considers it a valid implementation.

### The Mechanics of "Duck Typing"

> *"If it walks like a duck and quacks like a duck, it's a duck."*

This approach delivers clean decoupling. It allows consumers to define their own minimal interfaces instead of forcing producers to ship massive contract files, leading to a highly modular and extensible system architecture.

---

## 5. Real-World Deep Dive Lab: Building a Decoupled API Store Layer

To complete Day 2, let's design an enterprise-grade decoupled service layer. We will build a product service that interacts with a database repository.

By applying implicit interface contracts, we can decouple our business logic entirely from our storage layer, allowing us to swap a local mock database layer for a production MySQL instance with zero code changes.

### 🛠️ Execution Implementation Instructions

Ensure your workspace directory layout matches this structure:

```bash
mkdir -p internal/domain
```

### 📜 File: `internal/domain/product.go`

Create the domain file defining your data models, interface definitions, and core service operations.

```go
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
```

### 📜 File: `cmd/api/main.go`

Create your primary executable module. This file constructs a fast, memory-isolated mock database structure that satisfies the store interface automatically, allowing you to run your application instantly.

```go
package main

import (
	"errors"
	"fmt"
	"go-mysql-api/internal/domain"
	"os"
)

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
		fmt.Printf("Execution Error: %v
", err)
		os.Exit(1)
	}

	fmt.Println("🚀 Registration processing execution succeeded!")
	fmt.Printf("Assigned Row ID: %d
", product.ID)
	fmt.Printf("Stored Model Name: %s
", product.Name)
	fmt.Printf("Stored Model Price: %.2f
", product.Price)

	// 4. Test service validation rules with invalid inputs
	_, err = service.RegisterNewProduct("", -10.00)
	if err != nil {
		fmt.Printf("\nValidation intercept verification passed: %v
", err)
	}
}
```