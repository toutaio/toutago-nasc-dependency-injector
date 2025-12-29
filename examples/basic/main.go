// Package main demonstrates basic Nasc dependency injection usage
package main

import (
	"fmt"
	"log"

	nasc "github.com/toutaio/toutago-nasc-dependency-injector"
)

// Logger interface defines logging behavior
type Logger interface {
	Info(message string)
	Error(message string)
}

// ConsoleLogger implements Logger
type ConsoleLogger struct{}

func (l *ConsoleLogger) Info(message string) {
	fmt.Println("[INFO]", message)
}

func (l *ConsoleLogger) Error(message string) {
	fmt.Println("[ERROR]", message)
}

// Database interface defines database operations
type Database interface {
	Connect() error
	Query(sql string) ([]map[string]interface{}, error)
}

// MockDatabase implements Database
type MockDatabase struct {
	connected bool
}

func (db *MockDatabase) Connect() error {
	db.connected = true
	return nil
}

func (db *MockDatabase) Query(sql string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
	}, nil
}

// UserRepository handles user data access
type UserRepository interface {
	FindAll() ([]string, error)
}

// DBUserRepository implements UserRepository
type DBUserRepository struct {
	db     Database
	logger Logger
}

// NewDBUserRepository creates a repository with constructor injection
func NewDBUserRepository(db Database, logger Logger) *DBUserRepository {
	return &DBUserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *DBUserRepository) FindAll() ([]string, error) {
	r.logger.Info("Fetching all users from database")

	err := r.db.Connect()
	if err != nil {
		r.logger.Error("Failed to connect to database")
		return nil, err
	}

	results, err := r.db.Query("SELECT * FROM users")
	if err != nil {
		r.logger.Error("Failed to query users")
		return nil, err
	}

	var users []string
	for _, row := range results {
		if name, ok := row["name"].(string); ok {
			users = append(users, name)
		}
	}

	r.logger.Info(fmt.Sprintf("Found %d users", len(users)))
	return users, nil
}

// UserService provides user business logic
type UserService interface {
	GetAllUsers() ([]string, error)
}

// DefaultUserService implements UserService
type DefaultUserService struct {
	repo   UserRepository
	logger Logger
}

// NewUserService creates a service with constructor injection
func NewUserService(repo UserRepository, logger Logger) *DefaultUserService {
	return &DefaultUserService{
		repo:   repo,
		logger: logger,
	}
}

func (s *DefaultUserService) GetAllUsers() ([]string, error) {
	s.logger.Info("UserService: Getting all users")
	return s.repo.FindAll()
}

func main() {
	fmt.Println("=== Nasc Basic Example ===")

	// Example 1: Simple binding and resolution
	example1()

	fmt.Println()

	// Example 2: Constructor injection
	example2()

	fmt.Println()

	// Example 3: Transient lifetime
	example3()
}

func example1() {
	fmt.Println("--- Example 1: Simple Binding ---")

	// Create container
	container := nasc.New()

	// Bind logger interface to implementation
	err := container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err != nil {
		log.Fatal(err)
	}

	// Resolve logger
	logger := container.Make((*Logger)(nil)).(Logger)

	// Use logger
	logger.Info("This is a simple binding example")
	logger.Error("This is an error message")
}

func example2() {
	fmt.Println("--- Example 2: Constructor Injection ---")

	// Create container
	container := nasc.New()

	// Bind dependencies (using transient for now)
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	container.Bind((*Database)(nil), &MockDatabase{})

	// Create instances manually (constructor injection will be in later phases)
	logger := container.Make((*Logger)(nil)).(Logger)
	db := container.Make((*Database)(nil)).(Database)

	// Manual construction showing dependency injection pattern
	repo := NewDBUserRepository(db, logger)
	service := NewUserService(repo, logger)

	// Use service
	users, err := service.GetAllUsers()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Users:", users)
}

func example3() {
	fmt.Println("--- Example 3: Transient Lifetime ---")

	container := nasc.New()

	// Transient: new instance every time
	container.Bind((*Database)(nil), &MockDatabase{})

	db1 := container.Make((*Database)(nil)).(Database)
	db2 := container.Make((*Database)(nil)).(Database)

	fmt.Printf("Transient - Different instances? %v\n", db1 != db2)
	fmt.Println("(Each Make() call creates a new instance)")
}
