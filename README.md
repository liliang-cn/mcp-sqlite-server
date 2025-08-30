# MCP SQLite Server

A Model Context Protocol (MCP) server for SQLite database operations, built with Go. This server allows AI assistants to interact with SQLite databases through a secure, directory-restricted interface.

## Features

- **Multi-database support**: Switch between different SQLite databases dynamically
- **Directory security**: Operations restricted to specified allowed directories
- **Complete SQLite operations**: Query, execute, transactions, table management, indexing
- **Database management**: Create, delete, list, and switch between databases
- **Query analysis**: Analyze query execution plans and get database statistics
- **Transaction support**: Execute multiple statements atomically

## Installation

### Prerequisites

- Go 1.19 or later
- SQLite3

### Install via go install (recommended)

```bash
go install github.com/liliang-cn/mcp-sqlite-server@latest
```

### Build from source

```bash
git clone https://github.com/liliang-cn/mcp-sqlite-server.git
cd mcp-sqlite-server
go build -o mcp-sqlite-server
```

## Usage

### Basic usage

```bash
# Use default data directory
./mcp-sqlite-server

# Specify custom directories
./mcp-sqlite-server /path/to/db1 /path/to/db2
```

### With Claude Desktop

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "/path/to/mcp-sqlite-server",
      "args": ["/path/to/your/database/directory"]
    }
  }
}
```

## Available Tools

### Database Management

- `create_database` - Create new SQLite database files
- `switch_database` - Switch to different database
- `current_database` - Show current database path
- `list_database_files` - List all databases in directory
- `database_exists` - Check if database file exists
- `delete_database` - Delete database file (with confirmation)

### Table Operations

- `list_tables` - List all tables
- `describe_table` - Get table schema
- `create_table` - Create new tables
- `drop_table` - Delete tables

### Data Operations

- `query` - Execute SELECT queries
- `execute` - Execute INSERT/UPDATE/DELETE statements
- `transaction` - Execute multiple statements atomically

### Index Management

- `create_index` - Create indexes with advanced options
- `list_indexes` - List table indexes
- `drop_index` - Remove indexes

### Database Maintenance

- `vacuum` - Optimize database
- `analyze_query` - Analyze query execution plans
- `database_stats` - Get database statistics

## Security

- All operations are restricted to specified allowed directories
- Path validation prevents directory traversal attacks
- Database file validation ensures only SQLite files are accessed
- Transaction isolation ensures data consistency

## Examples

### Create and use a new database

```json
// Create database
{
  "name": "create_database",
  "arguments": {
    "directory": "/path/to/data",
    "purpose": "user management system"
  }
}

// Switch to the database
{
  "name": "switch_database",
  "arguments": {
    "db_path": "/path/to/data/user_management_system_1234.db"
  }
}
```

### Execute a transaction

```json
{
  "name": "transaction",
  "arguments": {
    "statements": [
      "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
      "INSERT INTO profiles (user_id, bio) VALUES (1, 'Software developer')",
      "UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = 1"
    ]
  }
}
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
