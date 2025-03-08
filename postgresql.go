package mcptools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

// PostgreSQLToolName is the name of the PostgreSQL tool
const PostgreSQLToolName = "postgresql_all_in_one"

// PostgreSQL represents a tool for performing PostgreSQL operations
type PostgreSQL struct {
	logger   observability.Logger
	config   PostgreSQLConfig
	connPool map[string]*sql.DB
	mu       sync.RWMutex
}

// PostgreSQLConfig represents the configuration for the PostgreSQL tool
type PostgreSQLConfig struct {
	DefaultDatabase string
	BlockedCommands []string
}

// DBConnection represents a PostgreSQL database connection configuration
type DBConnection struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgreSQL creates a new PostgreSQL tool with the given logger and configuration
func NewPostgreSQL(logger observability.Logger, config PostgreSQLConfig) *PostgreSQL {
	pg := &PostgreSQL{
		logger:   logger,
		config:   config,
		connPool: make(map[string]*sql.DB),
	}

	return pg
}

// PostgreSQLAllInOneTool remains mostly the same, but uses getConnection instead
func (p *PostgreSQL) PostgreSQLAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        PostgreSQLToolName,
		Description: "Performs PostgreSQL operations including querying, explaining queries, and retrieving schema information",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "operation": {
                    "type": "string",
                    "description": "Operation to perform (query, explain, schema, list_databases)",
                    "enum": ["query", "explain", "schema", "list_databases"]
                },
                "database": {
                    "type": "string",
                    "description": "Database identifier as configured in environment variables"
                },
                "query": {
                    "type": "string",
                    "description": "SQL query to execute (for query and explain operations)"
                },
                "table": {
                    "type": "string",
                    "description": "Table name (for schema operation)"
                }
            },
            "required": ["operation"]
        }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			span.SetAttributes(
				attribute.String("tool_name", params.Name),
				attribute.String("tool_argument", string(params.Arguments)),
			)
			defer span.End()

			p.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
			}).Info("Starting PostgreSQL operation")

			var input struct {
				Operation string `json:"operation"`
				Database  string `json:"database"`
				Query     string `json:"query"`
				Table     string `json:"table"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				p.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
			}

			// Handle list_databases operation first as it doesn't need a connection
			if input.Operation == "list_databases" {
				return p.listAvailableDatabases(), nil
			}

			// For all other operations, we need a database
			if input.Database == "" {
				return mcp.CallToolResult{}, fmt.Errorf("database identifier is required for operation: %s", input.Operation)
			}

			// Get database connection
			db, err := p.getConnection(input.Database)
			if err != nil {
				span.RecordError(err)
				return mcp.CallToolResult{}, err
			}

			switch input.Operation {
			case "query":
				if input.Query == "" {
					return mcp.CallToolResult{}, fmt.Errorf("query is required for operation 'query'")
				}
				return p.executeQuery(ctx, db, input.Query)

			case "explain":
				if input.Query == "" {
					return mcp.CallToolResult{}, fmt.Errorf("query is required for operation 'explain'")
				}
				return p.executeExplain(ctx, db, input.Query)

			case "schema":
				if input.Table == "" {
					return mcp.CallToolResult{}, fmt.Errorf("table is required for operation 'schema'")
				}
				return p.getTableSchema(ctx, db, input.Table)

			default:
				p.logger.WithFields(map[string]interface{}{
					"operation": input.Operation,
				}).Error("Invalid operation")
				return mcp.CallToolResult{}, fmt.Errorf("unknown operation: %s", input.Operation)
			}
		},
	}
}

func (p *PostgreSQL) executeQuery(ctx context.Context, db *sql.DB, query string) (mcp.CallToolResult, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return mcp.CallToolResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.CallToolResult{}, err
	}

	var result strings.Builder
	result.WriteString(strings.Join(columns, " | ") + "\n")
	result.WriteString(strings.Repeat("-", len(strings.Join(columns, " | "))) + "\n")

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.CallToolResult{}, err
		}

		var rowValues []string
		for _, val := range values {
			rowValues = append(rowValues, fmt.Sprintf("%v", val))
		}
		result.WriteString(strings.Join(rowValues, " | ") + "\n")
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

func (p *PostgreSQL) executeExplain(ctx context.Context, db *sql.DB, query string) (mcp.CallToolResult, error) {
	rows, err := db.QueryContext(ctx, "EXPLAIN ANALYZE "+query)
	if err != nil {
		return mcp.CallToolResult{}, err
	}
	defer rows.Close()

	var explain strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return mcp.CallToolResult{}, err
		}
		explain.WriteString(line + "\n")
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: explain.String(),
		}},
	}, nil
}

func (p *PostgreSQL) getTableSchema(ctx context.Context, db *sql.DB, tableName string) (mcp.CallToolResult, error) {
	query := `
        SELECT column_name, data_type, character_maximum_length, 
               is_nullable, column_default
        FROM information_schema.columns 
        WHERE table_name = $1
        ORDER BY ordinal_position;
    `

	rows, err := db.QueryContext(ctx, query, tableName)
	if err != nil {
		return mcp.CallToolResult{}, err
	}
	defer rows.Close()

	var schema strings.Builder
	schema.WriteString(fmt.Sprintf("Table: %s\n\n", tableName))
	schema.WriteString("Column Name | Data Type | Length | Nullable | Default\n")
	schema.WriteString("------------|-----------|---------|----------|----------\n")

	for rows.Next() {
		var (
			columnName, dataType, isNullable string
			maxLength                        sql.NullInt64
			defaultValue                     sql.NullString
		)
		if err := rows.Scan(&columnName, &dataType, &maxLength, &isNullable, &defaultValue); err != nil {
			return mcp.CallToolResult{}, err
		}

		schema.WriteString(fmt.Sprintf("%s | %s | %v | %s | %s\n",
			columnName,
			dataType,
			maxLength.Int64,
			isNullable,
			defaultValue.String))
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: schema.String(),
		}},
	}, nil
}

// New helper method to list available databases
func (p *PostgreSQL) listAvailableDatabases() mcp.CallToolResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var databases []string
	for dbName := range p.connPool {
		databases = append(databases, dbName)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: fmt.Sprintf("Available databases:\n%s", strings.Join(databases, "\n")),
		}},
	}
}

func (p *PostgreSQL) initializeConnections() error {
	var lastError error
	// Get all environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// Check if it's a database config variable
		if strings.HasSuffix(parts[0], "_DB_HOST") {
			dbName := strings.TrimSuffix(parts[0], "_DB_HOST")
			dbName = strings.ToLower(dbName)

			// Create connection config
			connConfig := DBConnection{
				Host:     os.Getenv(fmt.Sprintf("%s_DB_HOST", dbName)),
				Port:     os.Getenv(fmt.Sprintf("%s_DB_PORT", dbName)),
				User:     os.Getenv(fmt.Sprintf("%s_DB_USER", dbName)),
				Password: os.Getenv(fmt.Sprintf("%s_DB_PASSWORD", dbName)),
				DBName:   os.Getenv(fmt.Sprintf("%s_DB_NAME", dbName)),
				SSLMode:  os.Getenv(fmt.Sprintf("%s_DB_SSL_MODE", dbName)),
			}

			// Set defaults if not provided
			if connConfig.Port == "" {
				connConfig.Port = "5432"
			}
			if connConfig.SSLMode == "" {
				connConfig.SSLMode = "disable"
			}
			if connConfig.DBName == "" {
				connConfig.DBName = dbName
			}

			// Initialize the connection
			if err := p.initializeConnection(dbName, connConfig); err != nil {
				lastError = err
				p.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"database":                  dbName,
				}).Error("Failed to initialize database connection")
			}
		}
	}
	return lastError
}

func (p *PostgreSQL) initializeConnection(dbName string, config DBConnection) error {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	p.mu.Lock()
	p.connPool[dbName] = db
	p.mu.Unlock()

	return nil
}

// getConnection returns a connection to the specified database
func (p *PostgreSQL) getConnection(dbName string) (*sql.DB, error) {
	p.mu.RLock()
	db, exists := p.connPool[dbName]
	p.mu.RUnlock()

	if exists {
		// Test if connection is still alive
		if err := db.Ping(); err == nil {
			return db, nil
		}
		// Connection is dead, remove it
		p.mu.Lock()
		delete(p.connPool, dbName)
		p.mu.Unlock()
	}

	// Initialize connection if it doesn't exist or was dead
	if err := p.initializeConnections(); err != nil {
		return nil, fmt.Errorf("failed to initialize connections: %w", err)
	}

	p.mu.RLock()
	db, exists = p.connPool[dbName]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no connection found for database: %s", dbName)
	}

	return db, nil
}
