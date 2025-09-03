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
- SQLite3 (for creating/managing database files)

### Install via go install (recommended)

```bash
go install github.com/liliang-cn/mcp-sqlite-server@latest

# Add to PATH (for zsh users)
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Verify installation
mcp-sqlite-server --help
```

### Build from source

```bash
git clone https://github.com/liliang-cn/mcp-sqlite-server.git
cd mcp-sqlite-server
go build -o mcp-sqlite-server
```

## Usage

### Basic usage

**Required**: You must specify at least one database path or directory as an argument.

```bash
# Specify a single database file
mcp-sqlite-server /path/to/database.db

# Specify a directory containing .db files (will use the first found)
mcp-sqlite-server /path/to/db/directory

# Specify multiple directories for access control
mcp-sqlite-server /path/to/db/dir1 /path/to/db/dir2
```

**Note**: The server will exit with an error if:
- No arguments are provided
- No valid SQLite database files are found in the specified directories

### With Claude Desktop

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "mcp-sqlite-server",
      "args": ["/path/to/your/database/directory"]
    }
  }
}
```

**Important**: Make sure to:
1. Add `$HOME/go/bin` to your PATH (for zsh: add `export PATH="$HOME/go/bin:$PATH"` to `~/.zshrc`)
2. Provide a valid database file or directory path in the `args` array
3. Ensure the specified directory contains at least one `.db` file

## Available Tools (19 Total)

### Query & Data Manipulation
1. `query` - Execute a SELECT query on the SQLite database
2. `execute` - Execute an INSERT, UPDATE, or DELETE statement
3. `transaction` - Execute multiple SQL statements in a transaction (INSERT/UPDATE/DELETE only, no SELECT)

### Table Management
4. `create_table` - Create a new table in the database
5. `list_tables` - List all tables in the database
6. `describe_table` - Get the schema of a specific table
7. `drop_table` - Drop a table from the database

### Index Management
8. `create_index` - Create an index on a table column(s) with advanced options
9. `list_indexes` - List all indexes for a table
10. `drop_index` - Drop an index from the database

### Database Management
11. `create_database` - Create a new SQLite database file with an AI-generated name in the specified directory
12. `database_exists` - Check if a database file exists and is valid in allowed directories
13. `switch_database` - Switch to a different SQLite database file in allowed directories
14. `current_database` - Show the currently connected database file path
15. `list_database_files` - List all SQLite database files in a directory
16. `delete_database` - Delete a SQLite database file from allowed directories (CAUTION: This permanently deletes the file)

### Database Analysis & Optimization
17. `vacuum` - Optimize the database by rebuilding it
18. `analyze_query` - Analyze the execution plan of a SQL query
19. `database_stats` - Get database statistics and information

## Security

- All operations are restricted to specified allowed directories
- Path validation prevents directory traversal attacks
- Database file validation ensures only SQLite files are accessed
- Transaction isolation ensures data consistency

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
