package main

import (
	"log"
	"os"

	"github.com/liliang-cn/mcp-sqlite-server/database"
	"github.com/liliang-cn/mcp-sqlite-server/server"
)

func main() {
	// Check arguments
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <database_path_or_directory> [additional_directories...]\n", os.Args[0])
	}

	// Use all arguments as directory/file paths
	allowedDirs := os.Args[1:]

	// Find the first directory with databases
	var dbPath string

	for _, dir := range allowedDirs {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			// Check if directory has database files
			dbFiles, err := database.ListDatabaseFiles(dir)
			if err != nil {
				log.Printf("Warning: Failed to list database files in directory %s: %v", dir, err)
				continue
			}

			if len(dbFiles) > 0 {
				// Use the first database file found
				dbPath = dbFiles[0]
				log.Printf("Found %d database file(s) in directory %s, using: %s", len(dbFiles), dir, dbPath)
				break
			}
		}
	}

	// If no databases found, exit with error
	if dbPath == "" {
		log.Fatalf("No database files found in directories: %v. Please specify at least one SQLite database file.", allowedDirs)
	}

	// Create and start server with allowed directories
	srv, err := server.NewSQLiteServerWithDirs(dbPath, allowedDirs)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	log.Printf("MCP SQLite server starting with database: %s", dbPath)
	log.Printf("Allowed directories: %v", allowedDirs)

	// Start stdio server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
