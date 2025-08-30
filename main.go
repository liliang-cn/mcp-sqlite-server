package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/mcp-sqlite-server/database"
	"github.com/liliang-cn/mcp-sqlite-server/server"
)

func main() {
	// Get multiple directory paths from arguments
	var allowedDirs []string

	if len(os.Args) > 1 {
		// Use all arguments as directory paths
		allowedDirs = os.Args[1:]
	} else {
		// Default directory
		allowedDirs = []string{"./data"}
	}

	// Find the first directory with databases or use the first one as default
	var dbPath string
	var selectedDir string

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
				selectedDir = dir
				log.Printf("Found %d database file(s) in directory %s, using: %s", len(dbFiles), dir, dbPath)
				break
			}
		}
	}

	// If no databases found, use first directory and create default
	if dbPath == "" {
		selectedDir = allowedDirs[0]
		dbPath = filepath.Join(selectedDir, "mcp.db")
		log.Printf("No database files found, will use default: %s", dbPath)
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
