package main

import (
	"bootstrapper/internal/config"
	"fmt"
	"os"
)

func bootstrapDB() {
	fmt.Println("Initializing Day 1 microservice architecture...")

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

func main() {
	bootstrapDB()
}
