// Package main demonstrates Nasc DI in a web server context
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	nasc "github.com/toutaio/toutago-nasc-dependency-injector"
)

// Logger interface
type Logger interface {
	Info(message string)
	Error(message string)
}

// StructuredLogger implementation
type StructuredLogger struct {
	mu sync.Mutex
}

func (l *StructuredLogger) Info(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("[INFO] %s - %s\n", time.Now().Format(time.RFC3339), message)
}

func (l *StructuredLogger) Error(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("[ERROR] %s - %s\n", time.Now().Format(time.RFC3339), message)
}

// Database interface
type Database interface {
	Query(sql string) ([]map[string]interface{}, error)
	Execute(sql string) error
}

// InMemoryDatabase implementation
type InMemoryDatabase struct {
	data   []map[string]interface{}
	logger Logger
	mu     sync.RWMutex
}

func NewInMemoryDatabase(logger Logger) *InMemoryDatabase {
	return &InMemoryDatabase{
		logger: logger,
		data: []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		},
	}
}

func (db *InMemoryDatabase) Query(sql string) ([]map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	db.logger.Info(fmt.Sprintf("Executing query: %s", sql))
	return db.data, nil
}

func (db *InMemoryDatabase) Execute(sql string) error {
	db.logger.Info(fmt.Sprintf("Executing command: %s", sql))
	return nil
}

// UserRepository handles user data
type UserRepository interface {
	FindAll() ([]User, error)
	FindByID(id int) (*User, error)
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// DBUserRepository implementation
type DBUserRepository struct {
	db     Database
	logger Logger
}

func NewUserRepository(db Database, logger Logger) *DBUserRepository {
	return &DBUserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *DBUserRepository) FindAll() ([]User, error) {
	r.logger.Info("Repository: Finding all users")

	results, err := r.db.Query("SELECT * FROM users")
	if err != nil {
		return nil, err
	}

	var users []User
	for _, row := range results {
		user := User{
			ID:    int(row["id"].(int)),
			Name:  row["name"].(string),
			Email: row["email"].(string),
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *DBUserRepository) FindByID(id int) (*User, error) {
	r.logger.Info(fmt.Sprintf("Repository: Finding user %d", id))

	results, err := r.db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %d", id))
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	row := results[0]
	user := &User{
		ID:    int(row["id"].(int)),
		Name:  row["name"].(string),
		Email: row["email"].(string),
	}

	return user, nil
}

// UserHandler handles HTTP requests
type UserHandler struct {
	repo   UserRepository
	logger Logger
}

func NewUserHandler(repo UserRepository, logger Logger) *UserHandler {
	return &UserHandler{
		repo:   repo,
		logger: logger,
	}
}

func (h *UserHandler) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Handling GET /users request")

	users, err := h.repo.FindAll()
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get users: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Handling GET /user request")

	// Simple ID extraction (in real app, use router with params)
	var id int
	fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id)

	user, err := h.repo.FindByID(id)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		http.Error(w, "User Not Found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Middleware for request logging
func LoggingMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger.Info(fmt.Sprintf("Request: %s %s", r.Method, r.URL.Path))

			next.ServeHTTP(w, r)

			duration := time.Since(start)
			logger.Info(fmt.Sprintf("Completed in %v", duration))
		})
	}
}

func main() {
	fmt.Println("=== Nasc Web Server Example ===")

	// Create and configure container
	container := nasc.New()

	// Register dependencies (using current Phase 1 API)
	// Note: Later phases will add BindSingleton, BindConstructor, etc.
	container.Bind((*Logger)(nil), &StructuredLogger{})
	container.Bind((*Database)(nil), &InMemoryDatabase{})
	container.Bind((*UserRepository)(nil), &DBUserRepository{})
	container.Bind((*UserHandler)(nil), &UserHandler{})

	// Manually resolve and inject dependencies (demonstrates DI pattern)
	logger := container.Make((*Logger)(nil)).(Logger)
	db := NewInMemoryDatabase(logger)
	repo := NewUserRepository(db, logger)
	handler := NewUserHandler(repo, logger)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/users", handler.HandleGetUsers)
	mux.HandleFunc("/user", handler.HandleGetUser)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Apply middleware
	httpHandler := LoggingMiddleware(logger)(mux)

	logger.Info("Starting server on :8080")
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Try:")
	fmt.Println("  curl http://localhost:8080/users")

	if err := http.ListenAndServe(":8080", httpHandler); err != nil {
		log.Fatal("Server error:", err)
	}
}
