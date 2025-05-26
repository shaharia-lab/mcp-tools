package mcptools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shaharia-lab/goai"
	"github.com/stretchr/testify/mock"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgreSQL(t *testing.T) {
	logger := &MockLogger{}
	config := PostgreSQLConfig{
		DefaultDatabase: "test_db",
		BlockedCommands: []string{"DROP", "DELETE"},
	}

	pg := NewPostgreSQL(logger, config)

	assert.NotNil(t, pg)
	assert.Equal(t, config, pg.config)
	assert.NotNil(t, pg.connPool)
}

func TestPostgreSQL_PostgreSQLAllInOneTool(t *testing.T) {
	logger := &MockLogger{}
	pg := NewPostgreSQL(logger, PostgreSQLConfig{})
	tool := pg.PostgreSQLAllInOneTool()

	assert.Equal(t, PostgreSQLToolName, tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, tool.Handler)
}

func TestPostgreSQL_Query(t *testing.T) {
	// Create a sqlMock database connection
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create mock logger and set up expectations
	logger := new(MockLogger)

	// Only set up the expectations that are actually used
	logger.On("WithFields", mock.Anything).Return(logger)
	logger.On("Info", mock.Anything).Return()

	// Create PostgreSQL instance
	pg := NewPostgreSQL(logger, PostgreSQLConfig{})

	pg.mu.Lock()
	pg.connPool["test_db"] = db
	pg.mu.Unlock()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "test")
	sqlMock.ExpectQuery("SELECT").WillReturnRows(rows)

	input := map[string]interface{}{
		"operation": "query",
		"database":  "test_db",
		"query":     "SELECT * FROM test_table",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := pg.PostgreSQLAllInOneTool().Handler(
		context.Background(),
		goai.CallToolParams{
			Name:      PostgreSQLToolName,
			Arguments: inputJSON,
		},
	)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
	logger.AssertExpectations(t)
}

func TestPostgreSQL_InvalidOperation(t *testing.T) {
	// Create and set up mock logger
	logger := new(MockLogger)

	// Expect first WithFields call with tool info
	logger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
		return fields["tool_name"] != nil && fields["arguments"] != nil
	})).Return(logger)
	logger.On("Info", []interface{}{"Starting PostgreSQL operation"}).Return() // Fixed: expect slice of interface{}

	// Expect second WithFields call with operation info
	logger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
		return fields["operation"] == "invalid_operation"
	})).Return(logger)
	logger.On("Error", "Invalid operation").Return()

	pg := NewPostgreSQL(logger, PostgreSQLConfig{})

	input := map[string]interface{}{
		"operation": "invalid_operation",
		"database":  "test_db",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := pg.PostgreSQLAllInOneTool().Handler(
		context.Background(),
		goai.CallToolParams{
			Name:      PostgreSQLToolName,
			Arguments: inputJSON,
		},
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPostgreSQL_ListDatabases(t *testing.T) {
	// Create and set up mock logger
	logger := new(MockLogger)

	// First WithFields call with initial tool info
	logger.On("WithFields", map[string]interface{}{
		"tool_name": "postgresql",
		"arguments": "{\"operation\":\"list_databases\"}",
	}).Return(logger)

	// Second WithFields call for operation
	logger.On("WithFields", map[string]interface{}{
		"tool":      "postgresql",
		"operation": "listAvailableDatabases",
	}).Return(logger)

	// Third WithFields call with databases result
	logger.On("WithFields", map[string]interface{}{
		"tool":      "postgresql",
		"operation": "listAvailableDatabases",
		"databases": []string(nil),
	}).Return(logger)

	// Expect Info calls in the correct sequence
	logger.On("Info", []interface{}{"Starting PostgreSQL operation"}).Return()
	logger.On("Info", []interface{}{"Listing available databases"}).Return()
	logger.On("Info", []interface{}{"Databases listed successfully"}).Return()
	logger.On("Info", []interface{}{"PostgreSQL operation completed successfully"}).Return()

	// Create PostgreSQL instance with the mock logger
	pg := NewPostgreSQL(logger, PostgreSQLConfig{})

	input := map[string]interface{}{
		"operation": "list_databases",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := pg.PostgreSQLAllInOneTool().Handler(
		context.Background(),
		goai.CallToolParams{
			Name:      PostgreSQLToolName,
			Arguments: inputJSON,
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}
