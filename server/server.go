package server

import (
	"context"
	"fmt"

	"github.com/liliang-cn/mcp-sqlite-server/database"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type SQLiteServer struct {
	server      *server.MCPServer
	db          *database.SQLiteDB
	dbPath      string
	allowedDirs []string
}

// NewSQLiteServer creates a new SQLite MCP server
func NewSQLiteServer(dbPath string) (*SQLiteServer, error) {
	return NewSQLiteServerWithDirs(dbPath, []string{"./data"})
}

// NewSQLiteServerWithDirs creates a new SQLite MCP server with allowed directories
func NewSQLiteServerWithDirs(dbPath string, allowedDirs []string) (*SQLiteServer, error) {
	// Initialize database
	db, err := database.NewSQLiteDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create server instance
	srv := &SQLiteServer{
		db:          db,
		dbPath:      dbPath,
		allowedDirs: allowedDirs,
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-sqlite-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	srv.server = mcpServer

	// Register tool handlers
	srv.registerHandlers()

	return srv, nil
}

// NewSQLiteServerWithoutDB creates a new SQLite MCP server without an initial database
func NewSQLiteServerWithoutDB() *SQLiteServer {
	// Create server instance without database
	srv := &SQLiteServer{
		db:          nil,
		dbPath:      "",
		allowedDirs: []string{},
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-sqlite-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	srv.server = mcpServer

	// Register tool handlers (will work when database is set)
	srv.registerHandlers()

	return srv
}

// SetAllowedDirs sets the allowed directories for the server
func (s *SQLiteServer) SetAllowedDirs(dirs []string) {
	s.allowedDirs = dirs
}

// registerHandlers registers all tool handlers
func (s *SQLiteServer) registerHandlers() {
	// Add tools
	s.server.AddTool(mcp.Tool{
		Name:        "query",
		Description: "Execute a SELECT query on the SQLite database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL SELECT query to execute",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleQueryTool)

	s.server.AddTool(mcp.Tool{
		Name:        "execute",
		Description: "Execute an INSERT, UPDATE, or DELETE statement",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"statement": map[string]interface{}{
					"type":        "string",
					"description": "SQL statement to execute",
				},
			},
			Required: []string{"statement"},
		},
	}, s.handleExecuteTool)

	s.server.AddTool(mcp.Tool{
		Name:        "create_table",
		Description: "Create a new table in the database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table to create",
				},
				"columns": map[string]interface{}{
					"type":        "array",
					"description": "Array of column definitions",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Column name",
							},
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Column data type (INTEGER, TEXT, REAL, BLOB)",
							},
							"constraints": map[string]interface{}{
								"type":        "string",
								"description": "Optional constraints (PRIMARY KEY, NOT NULL, etc.)",
							},
						},
						"required": []string{"name", "type"},
					},
				},
			},
			Required: []string{"table_name", "columns"},
		},
	}, s.handleCreateTableTool)

	s.server.AddTool(mcp.Tool{
		Name:        "list_tables",
		Description: "List all tables in the database",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleListTablesTool)

	s.server.AddTool(mcp.Tool{
		Name:        "describe_table",
		Description: "Get the schema of a specific table",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table to describe",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleDescribeTableTool)

	s.server.AddTool(mcp.Tool{
		Name:        "transaction",
		Description: "Execute multiple SQL statements in a transaction (INSERT/UPDATE/DELETE only, no SELECT)",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"statements": map[string]interface{}{
					"type":        "array",
					"description": "Array of SQL statements to execute atomically (INSERT, UPDATE, DELETE only)",
					"items": map[string]interface{}{
						"type": "string",
					},
					"minItems": 1,
				},
			},
			Required: []string{"statements"},
		},
	}, s.handleTransactionTool)

	s.server.AddTool(mcp.Tool{
		Name:        "drop_table",
		Description: "Drop a table from the database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table to drop",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleDropTableTool)

	s.server.AddTool(mcp.Tool{
		Name:        "create_index",
		Description: "Create an index on a table column(s) with advanced options",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"index_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the index to create",
				},
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table",
				},
				"columns": map[string]interface{}{
					"type":        "array",
					"description": "Array of column specifications",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Column name",
							},
							"sort_order": map[string]interface{}{
								"type":        "string",
								"description": "Sort order (ASC or DESC)",
								"enum":        []string{"ASC", "DESC"},
							},
						},
						"required": []string{"name"},
					},
				},
				"unique": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the index should be unique",
				},
				"if_not_exists": map[string]interface{}{
					"type":        "boolean",
					"description": "Only create index if it doesn't already exist",
				},
				"where_clause": map[string]interface{}{
					"type":        "string",
					"description": "Optional WHERE clause for partial indexes",
				},
			},
			Required: []string{"index_name", "table_name", "columns"},
		},
	}, s.handleCreateIndexTool)

	s.server.AddTool(mcp.Tool{
		Name:        "list_indexes",
		Description: "List all indexes for a table",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table",
				},
			},
			Required: []string{"table_name"},
		},
	}, s.handleListIndexesTool)

	s.server.AddTool(mcp.Tool{
		Name:        "drop_index",
		Description: "Drop an index from the database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"index_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the index to drop",
				},
			},
			Required: []string{"index_name"},
		},
	}, s.handleDropIndexTool)

	s.server.AddTool(mcp.Tool{
		Name:        "vacuum",
		Description: "Optimize the database by rebuilding it",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleVacuum)

	s.server.AddTool(mcp.Tool{
		Name:        "analyze_query",
		Description: "Analyze the execution plan of a SQL query",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to analyze",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleAnalyzeQueryTool)

	s.server.AddTool(mcp.Tool{
		Name:        "database_stats",
		Description: "Get database statistics and information",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleDatabaseStatsTool)

	s.server.AddTool(mcp.Tool{
		Name:        "create_database",
		Description: "Create a new SQLite database file with an AI-generated name in the specified directory",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "Directory where the database should be created (must be in allowed directories)",
				},
				"purpose": map[string]interface{}{
					"type":        "string",
					"description": "Optional description of the database purpose (helps generate a suitable filename)",
				},
				"suggested_name": map[string]interface{}{
					"type":        "string",
					"description": "Optional suggested filename (without extension)",
				},
			},
			Required: []string{"directory"},
		},
	}, s.handleCreateDatabase)

	s.server.AddTool(mcp.Tool{
		Name:        "database_exists",
		Description: "Check if a database file exists and is valid in allowed directories",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the database file to check (must be in allowed directories)",
				},
			},
			Required: []string{"db_path"},
		},
	}, s.handleDatabaseExists)

	s.server.AddTool(mcp.Tool{
		Name:        "switch_database",
		Description: "Switch to a different SQLite database file in allowed directories",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the database file to switch to (must be in allowed directories)",
				},
			},
			Required: []string{"db_path"},
		},
	}, s.handleSwitchDatabase)

	s.server.AddTool(mcp.Tool{
		Name:        "current_database",
		Description: "Show the currently connected database file path",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleCurrentDatabase)

	s.server.AddTool(mcp.Tool{
		Name:        "list_database_files",
		Description: "List all SQLite database files in a directory",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "Directory to search for database files (required, must be in allowed directories)",
				},
			},
		},
	}, s.handleListDatabaseFiles)

	s.server.AddTool(mcp.Tool{
		Name:        "delete_database",
		Description: "Delete a SQLite database file from allowed directories (CAUTION: This permanently deletes the file)",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the database file to delete (must be in allowed directories)",
				},
				"confirm": map[string]interface{}{
					"type":        "boolean",
					"description": "Confirmation flag - must be true to actually delete the file",
				},
			},
			Required: []string{"db_path", "confirm"},
		},
	}, s.handleDeleteDatabase)
}

// Start starts the server
func (s *SQLiteServer) Start() error {
	return server.ServeStdio(s.server)
}

// Close closes the server and database connection
func (s *SQLiteServer) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Tool handler methods

// handleQueryTool handles query tool
func (s *SQLiteServer) handleQueryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}
	return s.handleQuery(ctx, args)
}

// handleExecuteTool handles execute tool
func (s *SQLiteServer) handleExecuteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}
	return s.handleExecute(ctx, args)
}

// handleCreateTableTool handles create table tool
func (s *SQLiteServer) handleCreateTableTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}
	return s.handleCreateTable(ctx, args)
}

// handleListTablesTool handles list tables tool
func (s *SQLiteServer) handleListTablesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleListTables(ctx)
}

// handleDescribeTableTool handles describe table tool
func (s *SQLiteServer) handleDescribeTableTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}
	return s.handleDescribeTable(ctx, args)
}

// handleTransactionTool handles transaction tool
func (s *SQLiteServer) handleTransactionTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}
	return s.handleTransaction(ctx, args)
}
