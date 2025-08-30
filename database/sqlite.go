package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db     *sql.DB
	dbPath string
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteDB{
		db:     db,
		dbPath: dbPath,
	}, nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// ExecuteQuery executes a SELECT query
func (s *SQLiteDB) ExecuteQuery(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Prepare result set
	var results []map[string]interface{}

	// Create interface{} slice for scanning
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	// Iterate through all rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create row mapping
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle []byte type (TEXT in SQLite)
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// ExecuteStatement executes INSERT/UPDATE/DELETE statements
func (s *SQLiteDB) ExecuteStatement(statement string, args ...interface{}) (int64, error) {
	result, err := s.db.Exec(statement, args...)
	if err != nil {
		return 0, fmt.Errorf("execution failed: %w", err)
	}

	// Return different results based on statement type
	upperStmt := strings.ToUpper(strings.TrimSpace(statement))
	if strings.HasPrefix(upperStmt, "INSERT") {
		return result.LastInsertId()
	}

	return result.RowsAffected()
}

// GetTables gets all table names
func (s *SQLiteDB) GetTables() ([]string, error) {
	query := `
		SELECT name FROM sqlite_master 
		WHERE type='table' 
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// GetTableSchema gets table structure
func (s *SQLiteDB) GetTableSchema(tableName string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	return s.ExecuteQuery(query)
}

// CreateTable creates a table
func (s *SQLiteDB) CreateTable(tableName string, columns []map[string]string) error {
	if len(columns) == 0 {
		return fmt.Errorf("no columns specified")
	}

	var columnDefs []string
	for _, col := range columns {
		name := col["name"]
		dataType := col["type"]
		constraints := col["constraints"]

		if name == "" || dataType == "" {
			return fmt.Errorf("column name and type are required")
		}

		def := fmt.Sprintf("%s %s", name, dataType)
		if constraints != "" {
			def += " " + constraints
		}
		columnDefs = append(columnDefs, def)
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
	_, err := s.db.Exec(createSQL)
	return err
}

// Transaction executes a transaction
func (s *SQLiteDB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DropTable drops a table
func (s *SQLiteDB) DropTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := s.db.Exec(query)
	return err
}

// CreateIndex creates an index on a table
func (s *SQLiteDB) CreateIndex(indexName, tableName string, columns []string, unique bool, ifNotExists bool) error {
	if len(columns) == 0 {
		return fmt.Errorf("at least one column must be specified")
	}

	var query string
	existsClause := ""
	if ifNotExists {
		existsClause = "IF NOT EXISTS "
	}

	uniqueClause := ""
	if unique {
		uniqueClause = "UNIQUE "
	}

	columnsStr := strings.Join(columns, ", ")
	query = fmt.Sprintf("CREATE %sINDEX %s%s ON %s (%s)",
		uniqueClause, existsClause, indexName, tableName, columnsStr)

	_, err := s.db.Exec(query)
	return err
}

// CreateIndexWithOptions creates an index with advanced options
func (s *SQLiteDB) CreateIndexWithOptions(options IndexOptions) error {
	if options.IndexName == "" {
		return fmt.Errorf("index name is required")
	}
	if options.TableName == "" {
		return fmt.Errorf("table name is required")
	}
	if len(options.Columns) == 0 {
		return fmt.Errorf("at least one column must be specified")
	}

	var parts []string
	parts = append(parts, "CREATE")

	if options.Unique {
		parts = append(parts, "UNIQUE")
	}

	parts = append(parts, "INDEX")

	if options.IfNotExists {
		parts = append(parts, "IF NOT EXISTS")
	}

	parts = append(parts, options.IndexName)
	parts = append(parts, "ON")
	parts = append(parts, options.TableName)

	// Build column specifications
	var columnSpecs []string
	for _, col := range options.Columns {
		spec := col.Name
		if col.SortOrder != "" {
			spec += " " + strings.ToUpper(col.SortOrder)
		}
		columnSpecs = append(columnSpecs, spec)
	}

	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(columnSpecs, ", ")))

	// Add WHERE clause if specified
	if options.WhereClause != "" {
		parts = append(parts, "WHERE")
		parts = append(parts, options.WhereClause)
	}

	query := strings.Join(parts, " ")
	_, err := s.db.Exec(query)
	return err
}

// IndexOptions represents options for creating an index
type IndexOptions struct {
	IndexName   string
	TableName   string
	Columns     []IndexColumn
	Unique      bool
	IfNotExists bool
	WhereClause string
}

// IndexColumn represents a column in an index
type IndexColumn struct {
	Name      string
	SortOrder string // "ASC" or "DESC"
}

// GetIndexes gets all indexes for a table with detailed information
func (s *SQLiteDB) GetIndexes(tableName string) ([]map[string]interface{}, error) {
	// First get all indexes for the table
	indexQuery := fmt.Sprintf(`
		SELECT name, sql
		FROM sqlite_master
		WHERE type='index'
		AND tbl_name='%s'
		AND name NOT LIKE 'sqlite_autoindex_%%'
	`, tableName)

	indexes, err := s.ExecuteQuery(indexQuery)
	if err != nil {
		return nil, err
	}

	// For each index, get detailed column information
	var detailedIndexes []map[string]interface{}
	for _, index := range indexes {
		indexName := index["name"].(string)

		// Get index info using PRAGMA index_info
		infoQuery := fmt.Sprintf("PRAGMA index_info(%s)", indexName)
		columns, err := s.ExecuteQuery(infoQuery)
		if err != nil {
			continue // Skip this index if we can't get info
		}

		// Get index list info for uniqueness
		listQuery := fmt.Sprintf("PRAGMA index_list(%s)", tableName)
		listInfo, err := s.ExecuteQuery(listQuery)
		if err != nil {
			continue
		}

		// Find if this index is unique
		isUnique := false
		for _, listItem := range listInfo {
			if listItem["name"] == indexName {
				if uniqueVal, ok := listItem["unique"]; ok {
					isUnique = uniqueVal == "1"
				}
				break
			}
		}

		// Build column list
		var columnNames []string
		for _, col := range columns {
			if colName, ok := col["name"]; ok {
				columnNames = append(columnNames, colName.(string))
			}
		}

		detailedIndex := map[string]interface{}{
			"name":        indexName,
			"columns":     columnNames,
			"unique":      isUnique,
			"sql":         index["sql"],
			"table_name":  tableName,
		}
		detailedIndexes = append(detailedIndexes, detailedIndex)
	}

	return detailedIndexes, nil
}

// DropIndex drops an index from the database
func (s *SQLiteDB) DropIndex(indexName string) error {
	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
	_, err := s.db.Exec(query)
	return err
}

// Vacuum optimizes the database
func (s *SQLiteDB) Vacuum() error {
	_, err := s.db.Exec("VACUUM")
	return err
}

// GetDatabaseStats gets database statistics
func (s *SQLiteDB) GetDatabaseStats() ([]map[string]interface{}, error) {
	return s.ExecuteQuery("PRAGMA database_list")
}

// AnalyzeQuery analyzes a query execution plan
func (s *SQLiteDB) AnalyzeQuery(query string) ([]map[string]interface{}, error) {
	analyzeQuery := fmt.Sprintf("EXPLAIN QUERY PLAN %s", query)
	return s.ExecuteQuery(analyzeQuery)
}

// CreateNewDatabase creates a new SQLite database file
func CreateNewDatabase(dbPath string) error {
	// Open database (this will create the file if it doesn't exist)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to new database: %w", err)
	}

	// Create a simple test table to verify the database is working
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS _mcp_init (
			id INTEGER PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			version TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create init table: %w", err)
	}

	// Insert initialization record
	_, err = db.Exec(`
		INSERT INTO _mcp_init (version) VALUES (?)
	`, "1.0.0")
	if err != nil {
		return fmt.Errorf("failed to insert init record: %w", err)
	}

	return nil
}

// DatabaseExists checks if a database file exists and is valid
func DatabaseExists(dbPath string) bool {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return false
	}

	// Just check if it's a valid SQLite database by querying sqlite_master
	// Don't require our specific _mcp_init table for existing databases
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	return err == nil
}

// SwitchDatabase switches to a different database file
func (s *SQLiteDB) SwitchDatabase(newDbPath string) error {
	// Close the current connection
	if s.db != nil {
		s.db.Close()
	}

	// Open new database connection
	db, err := sql.Open("sqlite3", newDbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Update the instance
	s.db = db
	s.dbPath = newDbPath

	return nil
}

// GetCurrentDatabasePath returns the current database path
func (s *SQLiteDB) GetCurrentDatabasePath() string {
	return s.dbPath
}

// ListDatabaseFiles lists all SQLite database files in the given directory
func ListDatabaseFiles(dirPath string) ([]string, error) {
	if dirPath == "" {
		dirPath = "."
	}

	files, err := filepath.Glob(filepath.Join(dirPath, "*.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to list database files: %w", err)
	}

	// Also check for .sqlite and .sqlite3 extensions
	sqliteFiles, _ := filepath.Glob(filepath.Join(dirPath, "*.sqlite"))
	sqlite3Files, _ := filepath.Glob(filepath.Join(dirPath, "*.sqlite3"))
	
	files = append(files, sqliteFiles...)
	files = append(files, sqlite3Files...)

	// Filter out files that are not valid SQLite databases
	var validDatabases []string
	for _, file := range files {
		if DatabaseExists(file) || isValidSQLiteFile(file) {
			validDatabases = append(validDatabases, file)
		}
	}

	return validDatabases, nil
}

// isValidSQLiteFile checks if a file is a valid SQLite database by checking the header
func isValidSQLiteFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// SQLite files start with "SQLite format 3\000"
	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil || n < 16 {
		return false
	}

	expectedHeader := "SQLite format 3\x00"
	return string(header) == expectedHeader
}

// DeleteDatabase deletes a database file from the filesystem
func DeleteDatabase(dbPath string) error {
	// Check if file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist: %s", dbPath)
	}
	
	// Try to delete the file
	if err := os.Remove(dbPath); err != nil {
		return fmt.Errorf("failed to delete database file: %w", err)
	}
	
	return nil
}
