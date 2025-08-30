package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/mcp-sqlite-server/database"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleCallTool handles tool call requests
func (s *SQLiteServer) handleCallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	switch request.Params.Name {
	case "query":
		return s.handleQuery(ctx, args)
	case "execute":
		return s.handleExecute(ctx, args)
	case "create_table":
		return s.handleCreateTable(ctx, args)
	case "list_tables":
		return s.handleListTables(ctx)
	case "describe_table":
		return s.handleDescribeTable(ctx, args)
	case "transaction":
		return s.handleTransaction(ctx, args)
	case "drop_table":
		return s.handleDropTableTool(ctx, request)
	case "create_index":
		return s.handleCreateIndexTool(ctx, request)
	case "list_indexes":
		return s.handleListIndexesTool(ctx, request)
	case "drop_index":
		return s.handleDropIndexTool(ctx, request)
	case "vacuum":
		return s.handleVacuum(ctx, request)
	case "analyze_query":
		return s.handleAnalyzeQueryTool(ctx, request)
	case "database_stats":
		return s.handleDatabaseStatsTool(ctx, request)
	case "create_database":
		return s.handleCreateDatabase(ctx, request)
	case "database_exists":
		return s.handleDatabaseExists(ctx, request)
	case "delete_database":
		return s.handleDeleteDatabase(ctx, request)
	default:
		return nil, fmt.Errorf("unknown tool: %s", request.Params.Name)
	}
}

// handleQuery handles query requests
func (s *SQLiteServer) handleQuery(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	// Validate that it's a SELECT query
	trimmedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(trimmedQuery, "SELECT") && !strings.HasPrefix(trimmedQuery, "PRAGMA") {
		return nil, fmt.Errorf("only SELECT and PRAGMA queries are allowed with this tool")
	}

	results, err := s.db.ExecuteQuery(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// 格式化结果
	jsonResult, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("[Database: %s]\nQuery executed successfully. Returned %d rows:\n%s",
					s.db.GetCurrentDatabasePath(), len(results), string(jsonResult)),
			},
		},
	}, nil
}

// handleExecute handles execute statement requests
func (s *SQLiteServer) handleExecute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	statement, ok := args["statement"].(string)
	if !ok {
		return nil, fmt.Errorf("statement parameter is required")
	}

	// Validate it's not a SELECT query
	trimmedStmt := strings.TrimSpace(strings.ToUpper(statement))
	if strings.HasPrefix(trimmedStmt, "SELECT") {
		return nil, fmt.Errorf("use the 'query' tool for SELECT statements")
	}

	affected, err := s.db.ExecuteStatement(statement)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	var message string
	if strings.HasPrefix(trimmedStmt, "INSERT") {
		message = fmt.Sprintf("Insert successful. Last insert ID: %d", affected)
	} else {
		message = fmt.Sprintf("Statement executed successfully. Rows affected: %d", affected)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleCreateTable handles create table requests
func (s *SQLiteServer) handleCreateTable(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	tableName, ok := args["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required")
	}

	columnsRaw, ok := args["columns"]
	if !ok {
		return nil, fmt.Errorf("columns parameter is required")
	}

	// Convert column definitions
	columnsArray, ok := columnsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("columns must be an array")
	}

	var columns []map[string]string
	for _, col := range columnsArray {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each column must be an object")
		}

		column := make(map[string]string)
		if name, ok := colMap["name"].(string); ok {
			column["name"] = name
		}
		if colType, ok := colMap["type"].(string); ok {
			column["type"] = colType
		}
		if constraints, ok := colMap["constraints"].(string); ok {
			column["constraints"] = constraints
		}

		columns = append(columns, column)
	}

	if err := s.db.CreateTable(tableName, columns); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Table '%s' created successfully", tableName),
			},
		},
	}, nil
}

// handleListTables handles list tables requests
func (s *SQLiteServer) handleListTables(ctx context.Context) (*mcp.CallToolResult, error) {
	tables, err := s.db.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	var message string
	if len(tables) == 0 {
		message = "No tables found in the database"
	} else {
		message = fmt.Sprintf("Found %d table(s):\n", len(tables))
		for _, table := range tables {
			message += fmt.Sprintf("- %s\n", table)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleDescribeTable handles describe table requests
func (s *SQLiteServer) handleDescribeTable(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	tableName, ok := args["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required")
	}

	schema, err := s.db.GetTableSchema(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}

	var message string
	if len(schema) == 0 {
		message = fmt.Sprintf("Table '%s' does not exist or has no columns", tableName)
	} else {
		// Format results
		jsonSchema, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format schema: %w", err)
		}
		message = fmt.Sprintf("Schema for table '%s':\n%s", tableName, string(jsonSchema))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleTransaction handles transaction requests
func (s *SQLiteServer) handleTransaction(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	statementsRaw, ok := args["statements"]
	if !ok {
		return nil, fmt.Errorf("statements parameter is required")
	}

	statementsArray, ok := statementsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("statements must be an array")
	}

	if len(statementsArray) == 0 {
		return nil, fmt.Errorf("at least one statement is required")
	}

	var statements []string
	for i, stmt := range statementsArray {
		if s, ok := stmt.(string); ok {
			// Validate that it's not a SELECT query
			trimmedStmt := strings.TrimSpace(strings.ToUpper(s))
			if strings.HasPrefix(trimmedStmt, "SELECT") {
				return nil, fmt.Errorf("statement %d: SELECT queries are not allowed in transactions, use the 'query' tool instead", i+1)
			}
			statements = append(statements, s)
		} else {
			return nil, fmt.Errorf("statement %d must be a string", i+1)
		}
	}

	var totalAffected int64
	var executedStatements int

	err := s.db.Transaction(func(tx *sql.Tx) error {
		for i, stmt := range statements {
			result, err := tx.Exec(stmt)
			if err != nil {
				return fmt.Errorf("statement %d (%s): %w", i+1, strings.Split(stmt, " ")[0], err)
			}

			if affected, err := result.RowsAffected(); err == nil {
				totalAffected += affected
			}
			executedStatements++
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	var message string
	if executedStatements == 1 {
		message = fmt.Sprintf("Transaction completed successfully. 1 statement executed. Rows affected: %d", totalAffected)
	} else {
		message = fmt.Sprintf("Transaction completed successfully. %d statements executed. Total rows affected: %d", executedStatements, totalAffected)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleDropTable handles drop table requests
func (s *SQLiteServer) handleDropTableTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	tableName, ok := args["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required")
	}

	if err := s.db.DropTable(tableName); err != nil {
		return nil, fmt.Errorf("failed to drop table: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Table '%s' dropped successfully", tableName),
			},
		},
	}, nil
}

// handleCreateIndex handles create index requests
func (s *SQLiteServer) handleCreateIndexTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	indexName, ok := args["index_name"].(string)
	if !ok {
		return nil, fmt.Errorf("index_name parameter is required")
	}

	tableName, ok := args["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required")
	}

	// Parse columns
	columnsRaw, ok := args["columns"]
	if !ok {
		return nil, fmt.Errorf("columns parameter is required")
	}

	columnsArray, ok := columnsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("columns must be an array")
	}

	if len(columnsArray) == 0 {
		return nil, fmt.Errorf("at least one column must be specified")
	}

	var columns []string
	var indexColumns []database.IndexColumn

	for _, colRaw := range columnsArray {
		colMap, ok := colRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each column must be an object")
		}

		colName, ok := colMap["name"].(string)
		if !ok {
			return nil, fmt.Errorf("column name is required")
		}

		columns = append(columns, colName)

		indexCol := database.IndexColumn{Name: colName}
		if sortOrder, ok := colMap["sort_order"].(string); ok {
			indexCol.SortOrder = sortOrder
		}
		indexColumns = append(indexColumns, indexCol)
	}

	// Parse optional parameters
	unique := false
	if uniqueVal, ok := args["unique"].(bool); ok {
		unique = uniqueVal
	}

	ifNotExists := false
	if ifNotExistsVal, ok := args["if_not_exists"].(bool); ok {
		ifNotExists = ifNotExistsVal
	}

	whereClause := ""
	if whereVal, ok := args["where_clause"].(string); ok {
		whereClause = whereVal
	}

	// Use advanced options if any advanced features are requested
	if len(indexColumns) > 1 || whereClause != "" || (len(indexColumns) == 1 && indexColumns[0].SortOrder != "") {
		options := database.IndexOptions{
			IndexName:   indexName,
			TableName:   tableName,
			Columns:     indexColumns,
			Unique:      unique,
			IfNotExists: ifNotExists,
			WhereClause: whereClause,
		}

		if err := s.db.CreateIndexWithOptions(options); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else {
		// Use simple method for single column, no sort order, no where clause
		if err := s.db.CreateIndex(indexName, tableName, columns, unique, ifNotExists); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Build response message
	indexType := "non-unique"
	if unique {
		indexType = "unique"
	}

	existsText := ""
	if ifNotExists {
		existsText = " (if not exists)"
	}

	response := fmt.Sprintf("%s index '%s'%s created successfully on %s.%s",
		indexType, indexName, existsText, tableName, strings.Join(columns, ", "))

	if whereClause != "" {
		response += fmt.Sprintf(" WHERE %s", whereClause)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

// handleListIndexes handles list indexes requests
func (s *SQLiteServer) handleListIndexesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	tableName, ok := args["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required")
	}

	indexes, err := s.db.GetIndexes(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	var message string
	if len(indexes) == 0 {
		message = fmt.Sprintf("No indexes found for table '%s'", tableName)
	} else {
		message = fmt.Sprintf("Found %d index(es) for table '%s':\n", len(indexes), tableName)
		for _, index := range indexes {
			if name, ok := index["name"].(string); ok {
				message += fmt.Sprintf("- %s\n", name)
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleDropIndexTool handles drop index requests
func (s *SQLiteServer) handleDropIndexTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	indexName, ok := args["index_name"].(string)
	if !ok {
		return nil, fmt.Errorf("index_name parameter is required")
	}

	err := s.db.DropIndex(indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to drop index '%s': %w", indexName, err)
	}

	message := fmt.Sprintf("Successfully dropped index '%s'", indexName)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// handleVacuum handles vacuum requests
func (s *SQLiteServer) handleVacuum(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.db.Vacuum(); err != nil {
		return nil, fmt.Errorf("failed to vacuum database: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Database vacuum completed successfully",
			},
		},
	}, nil
}

// handleAnalyzeQuery handles analyze query requests
func (s *SQLiteServer) handleAnalyzeQueryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	plan, err := s.db.AnalyzeQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	// Format the query plan
	jsonPlan, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format query plan: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Query execution plan:\n%s", string(jsonPlan)),
			},
		},
	}, nil
}

// handleDatabaseStats handles database stats requests
func (s *SQLiteServer) handleDatabaseStatsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := s.db.GetDatabaseStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	// Format the stats
	jsonStats, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format database stats: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Database statistics:\n%s", string(jsonStats)),
			},
		},
	}, nil
}

// handleCreateDatabase handles create database requests
func (s *SQLiteServer) handleCreateDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	directory, ok := args["directory"].(string)
	if !ok || directory == "" {
		return nil, fmt.Errorf("directory parameter is required and cannot be empty")
	}

	// Auto-replace current directory with first allowed directory
	if directory == "." || directory == "./" {
		if len(s.allowedDirs) > 0 {
			directory = s.allowedDirs[0]
		} else {
			return nil, fmt.Errorf("no allowed directories configured")
		}
	}

	// Validate directory
	if err := s.validateDirectory(directory); err != nil {
		return nil, err
	}

	// Generate filename based on purpose or use suggested name
	var filename string
	if suggestedName, ok := args["suggested_name"].(string); ok && suggestedName != "" {
		filename = suggestedName + ".db"
	} else if purpose, ok := args["purpose"].(string); ok && purpose != "" {
		// Generate filename based on purpose
		filename = generateFilenameFromPurpose(purpose)
	} else {
		// Default filename with timestamp
		filename = fmt.Sprintf("database_%d.db", time.Now().Unix())
	}

	// Construct full path
	dbPath := filepath.Join(directory, filename)

	// Check if file already exists
	if _, err := os.Stat(dbPath); err == nil {
		// File exists, generate unique name
		base := strings.TrimSuffix(filename, ".db")
		for i := 1; ; i++ {
			testPath := filepath.Join(directory, fmt.Sprintf("%s_%d.db", base, i))
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				dbPath = testPath
				filename = fmt.Sprintf("%s_%d.db", base, i)
				break
			}
		}
	}

	if err := database.CreateNewDatabase(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Database created successfully:\nPath: %s\nFilename: %s", dbPath, filename),
			},
		},
	}, nil
}

// handleDatabaseExists handles database exists check requests
func (s *SQLiteServer) handleDatabaseExists(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	dbPath, ok := args["db_path"].(string)
	if !ok {
		return nil, fmt.Errorf("db_path parameter is required")
	}

	// Validate that the database path is in an allowed directory
	if err := s.validateFilePath(dbPath); err != nil {
		return nil, err
	}

	exists := database.DatabaseExists(dbPath)
	status := "does not exist"
	if exists {
		status = "exists and is valid"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Database at %s %s", dbPath, status),
			},
		},
	}, nil
}

// handleSwitchDatabase handles switching to a different database file
func (s *SQLiteServer) handleSwitchDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	dbPath, ok := args["db_path"].(string)
	if !ok {
		return nil, fmt.Errorf("db_path parameter is required")
	}

	// Validate that the database path is in an allowed directory
	if err := s.validateFilePath(dbPath); err != nil {
		return nil, err
	}

	// Check if the database file exists
	if !database.DatabaseExists(dbPath) {
		return nil, fmt.Errorf("database file does not exist or is not a valid SQLite database: %s", dbPath)
	}

	// Switch to the new database
	if err := s.db.SwitchDatabase(dbPath); err != nil {
		return nil, fmt.Errorf("failed to switch database: %w", err)
	}

	// Update server's dbPath field
	s.dbPath = dbPath

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully switched to database: %s", dbPath),
			},
		},
	}, nil
}

// handleCurrentDatabase handles showing the current database path
func (s *SQLiteServer) handleCurrentDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	currentPath := s.db.GetCurrentDatabasePath()

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Currently connected to database: %s", currentPath),
			},
		},
	}, nil
}

// handleListDatabaseFiles handles listing database files in a directory
func (s *SQLiteServer) handleListDatabaseFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	directory, ok := args["directory"].(string)
	if !ok || directory == "" {
		return nil, fmt.Errorf("directory parameter is required and cannot be empty")
	}

	// Auto-replace current directory with first allowed directory
	if directory == "." || directory == "./" {
		if len(s.allowedDirs) > 0 {
			directory = s.allowedDirs[0]
		} else {
			return nil, fmt.Errorf("no allowed directories configured")
		}
	}

	// Validate directory
	if err := s.validateDirectory(directory); err != nil {
		return nil, err
	}

	databases, err := database.ListDatabaseFiles(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to list database files: %w", err)
	}

	var message string
	if len(databases) == 0 {
		message = fmt.Sprintf("No SQLite database files found in directory: %s", directory)
	} else {
		message = fmt.Sprintf("Found %d SQLite database file(s) in %s:\n", len(databases), directory)
		for _, db := range databases {
			message += fmt.Sprintf("- %s\n", db)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
	}, nil
}

// validateDirectory checks if the directory is in the allowed directories
func (s *SQLiteServer) validateDirectory(directory string) error {
	// Auto-replace current directory with first allowed directory
	if directory == "." || directory == "./" {
		if len(s.allowedDirs) > 0 {
			directory = s.allowedDirs[0]
		} else {
			return fmt.Errorf("no allowed directories configured")
		}
	}

	// Normalize directory path (remove trailing slash for comparison)
	normalizedDir := strings.TrimSuffix(directory, "/")

	// Check if directory is in allowed directories
	for _, allowedDir := range s.allowedDirs {
		normalizedAllowedDir := strings.TrimSuffix(allowedDir, "/")
		if normalizedDir == normalizedAllowedDir {
			return nil
		}
	}

	return fmt.Errorf("directory '%s' is not in allowed directories: %v", directory, s.allowedDirs)
}

// validateFilePath checks if the file path is in the allowed directories
func (s *SQLiteServer) validateFilePath(filePath string) error {
	// Check if file path is in any allowed directory
	for _, allowedDir := range s.allowedDirs {
		normalizedAllowedDir := strings.TrimSuffix(allowedDir, "/")
		if strings.HasPrefix(filePath, normalizedAllowedDir+"/") || strings.HasPrefix(filePath, normalizedAllowedDir) {
			return nil
		}
	}

	return fmt.Errorf("file path '%s' is not in allowed directories: %v", filePath, s.allowedDirs)
}

// generateFilenameFromPurpose creates a suitable filename based on the database purpose
func generateFilenameFromPurpose(purpose string) string {
	// Convert purpose to a valid filename
	purpose = strings.ToLower(purpose)
	purpose = strings.ReplaceAll(purpose, " ", "_")
	purpose = strings.ReplaceAll(purpose, "-", "_")

	// Remove special characters
	var result strings.Builder
	for _, r := range purpose {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	filename := result.String()
	if filename == "" {
		filename = "database"
	}

	// Limit length and add timestamp for uniqueness
	if len(filename) > 20 {
		filename = filename[:20]
	}

	return fmt.Sprintf("%s_%d.db", filename, time.Now().Unix()%10000)
}

// handleDeleteDatabase handles deleting a database file
func (s *SQLiteServer) handleDeleteDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments type")
	}

	dbPath, ok := args["db_path"].(string)
	if !ok {
		return nil, fmt.Errorf("db_path parameter is required")
	}

	confirm, ok := args["confirm"].(bool)
	if !ok || !confirm {
		return nil, fmt.Errorf("confirm parameter must be true to delete the database")
	}

	// Validate that the database path is in an allowed directory
	if err := s.validateFilePath(dbPath); err != nil {
		return nil, err
	}

	// Check if this is the currently connected database
	if dbPath == s.dbPath {
		return nil, fmt.Errorf("cannot delete the currently connected database. Please switch to another database first")
	}

	// Delete the database file
	if err := database.DeleteDatabase(dbPath); err != nil {
		return nil, fmt.Errorf("failed to delete database: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Database successfully deleted: %s", dbPath),
			},
		},
	}, nil
}
