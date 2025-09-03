package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/mcp-sqlite-server/database"
	"github.com/liliang-cn/mcp-sqlite-server/server"
)

func main() {
	// Define command line flags
	help := flag.Bool("help", false, "Show help message")
	h := flag.Bool("h", false, "Show help message (shorthand)")
	ver := flag.Bool("version", false, "Show version information")
	v := flag.Bool("v", false, "Show version information (shorthand)")
	
	flag.Parse()
	
	// Handle help flag
	if *help || *h {
		fmt.Printf("MCP SQLite Server v%s\n", Version)
		fmt.Println("A Model Context Protocol server for SQLite database operations")
		fmt.Println()
		fmt.Printf("Usage: %s [options] <database_path_or_directory> [additional_directories...]\n\n", os.Args[0])
		fmt.Println("Arguments:")
		fmt.Println("  database_path_or_directory  Path to SQLite database file or directory containing .db files")
		fmt.Println("  additional_directories      Optional additional directories for multi-database access")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help     Show this help message")
		fmt.Println("  -v, --version  Show version information")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Single database file")
		fmt.Println("  mcp-sqlite-server /path/to/database.db")
		fmt.Println()
		fmt.Println("  # Directory containing database files")
		fmt.Println("  mcp-sqlite-server /path/to/db/directory")
		fmt.Println()
		fmt.Println("  # Multiple directories for access control")
		fmt.Println("  mcp-sqlite-server /dir1 /dir2 /dir3")
		fmt.Println()
		fmt.Println("Features:")
		fmt.Println("  • Multi-database support with dynamic switching")
		fmt.Println("  • Directory security with path validation")
		fmt.Println("  • Complete SQLite operations (query, execute, transactions)")
		fmt.Println("  • Table and index management")
		fmt.Println("  • Database optimization and analysis tools")
		fmt.Println("  • 19 specialized tools for database operations")
		fmt.Println()
		fmt.Println("For more information, visit: https://github.com/liliang-cn/mcp-sqlite-server")
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
	
	// Check arguments
	if len(args) < 1 {
		log.Fatalf("Error: No database path specified.\nUsage: %s <database_path_or_directory> [additional_directories...]\nTry '%s --help' for more information.\n", os.Args[0], os.Args[0])
	}

	// Use all arguments as directory/file paths
	allowedDirs := args

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
