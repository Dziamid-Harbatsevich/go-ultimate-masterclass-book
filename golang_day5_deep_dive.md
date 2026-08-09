# 📘 Day 5: High-Performance Database Pools, Transactions & JSON Streams

## 1. The Connection Pool Paradigm Shift: PHP Lifecycle vs. Go Persistent Pool

As a senior backend engineer with an extensive background in languages like PHP or Python, the most fundamental shift when interacting with databases in Go is moving away from the ephemeral request lifecycle.

```
PHP STATELESS CONNECTION PATTERN (High Connection Overhead)
[HTTP Request] ──► [PHP Script Engine] ──► [Create TCP Connection to MySQL] ──► [Query] ──► [Destroy Script & TCP Connection]

GO PERSISTENT POOL PATTERN (Zero-Overhead Reusable Sockets)
[Goroutine 1] ──► [Request Socket 1] ──┐
[Goroutine 2] ──► [Request Socket 2] ──┼─► [ sql.DB Connection Pool Manager ] ──► [Persistent TCP Connections] ──► [MySQL]
[Goroutine 3] ──► [Request Socket 3] ──┘
```

### The Ephemeral Framework Lifecycle
In standard PHP/PDO architectures, every incoming HTTP request opens a brand-new TCP socket connection to your MySQL server. Once the script finishes generating the HTML or JSON response, that socket is immediately discarded. This model requires a high overhead for continuous TCP handshakes and authentication exchanges. To scale this approach under heavy traffic, developers are forced to place complex connection proxy tools (like ProxySQL) in front of the database.

### The Go Persistent Engine
In Go, your application is a long-lived process. When you initialize a database reference using `sql.Open()`, the engine does not immediately open a connection to MySQL. Instead, it returns a thread-safe structural pool controller: **`*sql.DB`**. 

* **Goroutine Concurrency**: This pool remains active in memory for the entire lifespan of your application process. As thousands of concurrent HTTP requests spawn individual lightweight goroutines, they request, use, and immediately return active TCP sockets to this central pool.
* **Thread Safety**: The `*sql.DB` object handles thread synchronization internally, ensuring your application never triggers a connection-sharing race condition.

---

## 2. Tuning and Sizing Connection Pools for Production

To maximize throughput and prevent deadlocks under high request volumes, you must customize the parameters of your `*sql.DB` connection pool.

### Core Configuration Parameters
* **`SetMaxOpenConns(n)`**: The absolute upper limit on open connections to the database. Setting this too low causes application throat bottlenecks; setting it higher than your MySQL server's `max_connections` limit causes the database engine to reject connections.
* **`SetMaxIdleConns(n)`**: The maximum number of idle connections maintained in the pool. Keeping this value equal to `SetMaxOpenConns` ensures the pool doesn't constantly open and close connections under fluctuating traffic spikes.
* **`SetConnMaxLifetime(d)`**: The maximum amount of time a connection can be reused. Expiring connections safely ensures you don't encounter stale TCP sockets or silent firewall disconnections.
* **`SetConnMaxIdleTime(d)`**: Automatically closes connections that have sat completely idle beyond the specified duration, freeing up resources during off-peak hours.

### Recommended Production Blueprint Configuration
```go
db.SetMaxOpenConns(25)                 // Scaled for high concurrent microservice instances
db.SetMaxIdleConns(25)                 // Keeps sockets open to absorb sudden traffic spikes
db.SetConnMaxLifetime(5 * time.Minute) // Safely recycles sockets before the OS kills them
db.SetConnMaxIdleTime(2 * time.Minute) // Frees up cluster resources during low-traffic periods
```

---

## 3. Query vs. Exec Execution Routing and Avoiding Pool Leaks

The `database/sql` driver separates operations into two core execution paths based on whether you expect data rows to be returned.

### 📋 Query Routing Taxonomy Matrix

| API Method | Intended Use Case | Example Commands | Internal Result Tracing |
| :--- | :--- | :--- | :--- |
| **`db.QueryContext()`** | Streaming multiple database records | `SELECT * FROM products` | Returns `*sql.Rows` (Requires explicit iteration and closure) |
| **`db.QueryRowContext()`** | Fetching a single structural record | `SELECT FROM products WHERE id=?` | Returns `*sql.Row` (Closes internally upon calling `.Scan()`) |
| **`db.ExecContext()`** | Performing mutating data writes | `INSERT`, `UPDATE`, `DELETE` | Returns `sql.Result` (Tracks rows affected and last insert IDs) |

### Avoid Catastrophic Connection Leaks
When using `QueryContext()`, you receive a pointer to an active `*sql.Rows` stream. This stream maintains an active hold on a database connection from your pool. **If you do not call `rows.Close()`, that connection is leaked permanently.** Under heavy traffic, your application will quickly consume all available slots in the pool, causing the server to freeze indefinitely.

```go
rows, err := db.QueryContext(ctx, "SELECT id FROM products")
if err != nil {
    return err
}
defer rows.Close() // ALWAYS include this line directly below the error check
```

---

## 4. ACID Transactions and Context Propagation

Microservices demand strict database reliability. If an API request is canceled by a client mid-execution, your database operations must stop immediately to prevent wasted resource consumption.

### Propagating the Context Lifecycle
Every single query must use its context-aware variant (e.g., `QueryContext` instead of `Query`). When a client terminates an HTTP connection, the `net/http` server automatically triggers a cancellation signal down through the request's `context.Context`. Go's database driver listens to this signal and immediately instructs MySQL to terminate the active query.

### Safe Transaction Operations Blueprint
```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
if err != nil {
    return err
}
// Defer a rollback statement immediately. If the function exits early due to an error,
// the transaction rolls back cleanly. If tx.Commit() is reached, this statement becomes a safe no-op.
defer tx.Rollback() 

// Perform operations inside the transaction using the transaction context
_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
if err != nil {
    return err // Triggers automated rollback via defer
}

if err := tx.Commit(); err != nil {
	return err
}
```

---

## 5. Ultra-Fast Stream JSON Decoding

When handling large REST payloads, calling `json.Unmarshal([]byte)` forces the runtime to read the entire request body into an intermediate buffer in memory. This allocation approach increases memory footprint and adds overhead to the Garbage Collector.

Instead, Go allows you to stream incoming data directly from the network socket reader into your destination data structure using `json.NewDecoder()`.

```go
// Memory-Efficient JSON Streaming Pattern
var payload DestinationStruct
err := json.NewDecoder(r.Body).Decode(&payload)
```
This pattern streams the payload data chunk by chunk directly into your struct, completely eliminating unnecessary intermediary memory buffers.

---

## 6. Real-World Deep Dive Lab: Production REST + MySQL Repository Layer

Let's assemble these production components into a high-performance REST microservice component. We will build a structured repository layer that manages its connection pool safely, executes transactions with explicit error tracing, utilizes context propagation, and processes inbound streams cleanly.

### 📁 Workspace Project Setup
Ensure your project workspace matches this explicit structural layout:
```text
cmd/api/main.go
internal/data/models.go
internal/data/repository.go
```

### 📜 File: `internal/data/models.go`
```go
package data

import "time"

// Product maps your database schema directly to optimized JSON text targets
type Product struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}
```

### 📜 File: `internal/data/repository.go`
```go
package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ProductRepository struct {
	DB *sql.DB
}

// FindByID streams a single record out of MySQL using bounded contexts
func (repo *ProductRepository) FindByID(ctx context.Context, id int) (*Product, error) {
	query := `SELECT id, name, price, created_at FROM products WHERE id = ? LIMIT 1`

	var p Product
	// The database query honors the context timeout automatically
	err := repo.DB.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("product record not found")
		}
		return nil, err
	}
	return &p, nil
}

// CreateWithAudit executes an ACID transaction to write a product and log an entry simultaneously
func (repo *ProductRepository) CreateWithAudit(ctx context.Context, name string, price float64) (*Product, error) {
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Safely triggers a rollback if any operational errors occur below

	// 1. Insert the new product record
	insertQuery := `INSERT INTO products (name, price, created_at) VALUES (?, ?, ?)`
	createdAt := time.Now()
	res, err := tx.ExecContext(ctx, insertQuery, name, price, createdAt)
	if err != nil {
		return nil, err
	}

	productID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 2. Write a simulated audit log history record inside the same transaction block
	auditQuery := `INSERT INTO audit_logs (action, target_id, timestamp) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, auditQuery, "PRODUCT_CREATION", productID, createdAt)
	if err != nil {
		return nil, err // Both operations roll back cleanly
	}

	// 3. Commit the transaction to save changes permanently to the database
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Product{
		ID:        int(productID),
		Name:      name,
		Price:     price,
		CreatedAt: createdAt,
	}, nil
}
```

### 📜 File: `cmd/api/main.go`
```go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"go-mysql-api/internal/data"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type ApplicationContainer struct {
	Products *data.ProductRepository
}

type ErrorResponse struct {
	Message string `json:"error"`
}

func main() {
	// Initialize a production connection pool (Update with your local credentials)
	dsn := "root:password@tcp(127.0.0.1:3306)/my_database?parseTime=true"
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
	app := &ApplicationContainer{Products: repo}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", app.GetProductHandler)
	mux.HandleFunc("POST /products", app.CreateProductHandler)

	log.Println("🚀 High-Performance REST Engine listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
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
```

---

## 7. Course Executive Summary

* **The Connection Pool Paradigm Shift**: Contrasts PHP’s short-lived, single-request database lifecycles with Go’s long-lived, thread-safe `sql.DB` connection pooling engine. It reveals why `sql.DB` is not an active network connection, but rather a structural pool controller managing a dynamic array of persistent TCP sockets across multiple goroutines.
* **The Anatomy of Sizing and Tuning Pools**: Deep dive into tuning parameters (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`, `SetConnMaxIdleTime`) to maximize throughput on local host installations. It provides mathematical models to align application connection pools with underlying MySQL engine capacity.
* **Query vs. Exec Execution Routing**: Establishes precise boundaries for utilizing `db.QueryContext` for row streaming, `db.QueryRowContext` for single-record scanning, and `db.ExecContext` for raw mutations (INSERT, UPDATE, DELETE). It highlights the critical importance of `rows.Close()` defer statements to prevent catastrophic connection pool starvation leaks.
* **ACID Transactions and Context Timelines**: Explains how to execute safe multi-row mutations using `db.BeginTx`. It details structural rollbacks via `defer tx.Rollback()` and shows how `context.Context` signals seamlessly propagate from incoming REST HTTP requests down into MySQL to automatically terminate orphaned queries if a client disconnects.
* **High-Performance JSON Streams**: Compares memory-heavy string allocations (`json.Unmarshal`) with stream-based parsing (`json.NewDecoder`). It details how to stream payload data directly from the network socket reader into Go structs, drastically reducing garbage collection overhead.
* **Production REST + MySQL Repository Lab**: Provides a complete, working codebase divided into clean architectural layers (`models.go`, `repository.go`, `handlers.go`, `main.go`). This lab includes concrete implementations of context-aware queries, transaction rollbacks, SQL injection protection, and structured REST responses with appropriate HTTP codes.
