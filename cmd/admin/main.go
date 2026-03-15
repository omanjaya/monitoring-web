package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

func main() {
	logger.Init()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to database
	db, err := mysql.Connect(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repository
	userRepo := mysql.NewUserRepository(db)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===========================================")
	fmt.Println("  Create Admin User - Monitoring Website")
	fmt.Println("===========================================")
	fmt.Println()

	// Get username
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// Check if user exists
	existing, _ := userRepo.GetByUsername(context.Background(), username)
	if existing != nil {
		fmt.Println("Error: Username sudah digunakan")
		os.Exit(1)
	}

	// Get email
	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	existingEmail, _ := userRepo.GetByEmail(context.Background(), email)
	if existingEmail != nil {
		fmt.Println("Error: Email sudah digunakan")
		os.Exit(1)
	}

	// Get full name
	fmt.Print("Full Name: ")
	fullName, _ := reader.ReadString('\n')
	fullName = strings.TrimSpace(fullName)

	// Get phone (optional)
	fmt.Print("Phone (optional): ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	// Get password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		os.Exit(1)
	}
	password := string(passwordBytes)
	fmt.Println()

	// Confirm password
	fmt.Print("Confirm Password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		os.Exit(1)
	}
	confirmPassword := string(confirmBytes)
	fmt.Println()

	if password != confirmPassword {
		fmt.Println("Error: Password tidak cocok")
		os.Exit(1)
	}

	if len(password) < 8 {
		fmt.Println("Error: Password minimal 8 karakter")
		os.Exit(1)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		os.Exit(1)
	}

	// Create user
	userCreate := &domain.UserCreate{
		Username: username,
		Email:    email,
		FullName: fullName,
		Phone:    phone,
	}

	id, err := userRepo.Create(context.Background(), userCreate, string(hashedPassword), "super_admin")
	if err != nil {
		fmt.Println("Error creating user:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("===========================================")
	fmt.Printf("  Admin user berhasil dibuat! (ID: %d)\n", id)
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("Anda sekarang dapat login dengan:")
	fmt.Printf("  Username: %s\n", username)
	fmt.Println("  Password: ********")
}
