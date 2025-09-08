package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/mcp-sqlite-server/database"
	"github.com/liliang-cn/mcp-sqlite-server/server"
	"strings"
)

func isDBFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".db") ||
		strings.HasSuffix(strings.ToLower(path), ".sqlite") ||
		strings.HasSuffix(strings.ToLower(path), ".sqlite3") ||
		strings.HasSuffix(strings.ToLower(path), ".db3")
}

func main() {
	// Define command line flags
	help := flag.Bool("help", false, "Show help message")
	h := flag.Bool("h", false, "Show help message (shorthand)")
	ver := flag.Bool("version", false, "Show version information")
	v := flag.Bool("v", false, "Show version information (shorthand)")
	
	flag.Parse()
	
	// Handle help flag
	if *help || *h {
		fmt.Printf("Usage: mcp-sqlite-server [database-path-or-directory] [additional-directories...]\n")
		fmt.Println("Note: Database paths can be provided via:")
		fmt.Println("  1. Command-line arguments (shown above)")
		fmt.Println("  2. MCP roots protocol (if client supports it)")
		fmt.Println("At least one database or directory must be provided by EITHER method for the server to operate.")
		os.Exit(0)
	}
	
	// Handle version flag
	if *ver || *v {
		fmt.Printf("mcp-sqlite-server version %s\n", Version)
		if BuildDate != "" {
			fmt.Printf("Build date: %s\n", BuildDate)
		}
		os.Exit(0)
	}
	
	// Get remaining arguments after flags
	args := flag.Args()
	
	// Print startup message
	fmt.Fprintln(os.Stderr, "Secure MCP SQLite Server running on stdio")
	
	// Check if arguments provided
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Started without database paths - waiting for client to provide roots via MCP protocol")
		// Start server without initial database, waiting for roots
		srv := server.NewSQLiteServerWithoutDB()
		defer srv.Close()
		
		// Start stdio server
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
		return
	}

	// Use all arguments as directory/file paths
	allowedDirs := args
	fmt.Fprintf(os.Stderr, "Starting with allowed directories: %v\n", allowedDirs)

	// Find the first directory with databases or database file
	var dbPath string
	var foundDatabases []string

	for _, path := range allowedDirs {
		stat, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Cannot access path %s: %v\n", path, err)
			continue
		}
		
		if stat.IsDir() {
			// Check if directory has database files
			dbFiles, err := database.ListDatabaseFiles(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to list database files in directory %s: %v\n", path, err)
				continue
			}

			if len(dbFiles) > 0 {
				foundDatabases = append(foundDatabases, dbFiles...)
				if dbPath == "" {
					// Use the first database file found
					dbPath = dbFiles[0]
					fmt.Fprintf(os.Stderr, "Found %d database file(s) in directory %s\n", len(dbFiles), path)
				}
			}
		} else if isDBFile(path) {
			// Direct database file path
			foundDatabases = append(foundDatabases, path)
			if dbPath == "" {
				dbPath = path
			}
		}
	}

	// If no databases found, start without initial database
	if dbPath == "" {
		fmt.Fprintf(os.Stderr, "No database files found in specified paths. Server will wait for database selection via MCP protocol.\n")
		srv := server.NewSQLiteServerWithoutDB()
		srv.SetAllowedDirs(allowedDirs)
		defer srv.Close()
		
		// Start stdio server
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
		return
	}

	// Create and start server with allowed directories
	srv, err := server.NewSQLiteServerWithDirs(dbPath, allowedDirs)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	fmt.Fprintf(os.Stderr, "Using database: %s\n", dbPath)
	if len(foundDatabases) > 1 {
		fmt.Fprintf(os.Stderr, "Additional databases available: %d\n", len(foundDatabases)-1)
	}

	// Start stdio server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
