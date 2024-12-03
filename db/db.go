package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func checkEnvVars(vars ...string) error {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			return fmt.Errorf("environment variable %s is not set", v)
		}
	}
	return nil
}

func InitDB() {
	requiredVars := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	if err := checkEnvVars(requiredVars...); err != nil {
		log.Fatalf("Database connection information is incomplete: %v", err)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s search_path=public sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	log.Printf("Connecting to database: %s", os.Getenv("DB_NAME"))

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("Database connection established")
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}
	return string(hashed), nil
}

func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func AuthenticateUser(username, password string) (bool, error) {
	log.Printf("Authenticating user: %s", username)

	var hashedPassword string
	query := "SELECT password FROM users WHERE username = $1"
	log.Printf("Executing query: %s with username: %q", query, username)

	err := DB.QueryRow(query, username).Scan(&hashedPassword)
	if err == sql.ErrNoRows {
		log.Printf("User %s not found in database", username)
		return false, nil
	} else if err != nil {
		log.Printf("Database query error: %v", err)
		return false, fmt.Errorf("database query error: %v", err)
	}

	log.Printf("Retrieved hashed password for user %s", username)
	return true, nil
}

func RegisterUser(username, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", username, hashedPassword)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("username %s already exists", username)
		}
		return fmt.Errorf("failed to register user: %v", err)
	}

	log.Printf("User %s registered successfully", username)
	return nil
}

func GetAllUsers() ([]string, error) {
	rows, err := DB.Query("SELECT username FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %v", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, fmt.Errorf("failed to scan user: %v", err)
		}
		users = append(users, username)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over users: %v", err)
	}

	log.Printf("Fetched %d users", len(users))
	return users, nil
}
