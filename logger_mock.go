package mcptools

import (
	"context"

	"github.com/shaharia-lab/goai"
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of goai.Logger
type MockLogger struct {
	mock.Mock
}

// Debugf logs a formatted debug message.
func (m *MockLogger) Debugf(format string, args ...interface{}) { m.Called(format, args) }

// Infof logs a formatted info message.
func (m *MockLogger) Infof(format string, args ...interface{}) { m.Called(format, args) }

// Warnf logs a formatted warning message.
func (m *MockLogger) Warnf(format string, args ...interface{}) { m.Called(format, args) }

// Errorf logs a formatted error message.
func (m *MockLogger) Errorf(format string, args ...interface{}) { m.Called(format, args) }

// Fatalf logs a formatted fatal message.
func (m *MockLogger) Fatalf(format string, args ...interface{}) { m.Called(format, args) }

// Panicf logs a formatted panic message.
func (m *MockLogger) Panicf(format string, args ...interface{}) { m.Called(format, args) }

// Debug level logging methods

// Debug logs a debug message.
func (m *MockLogger) Debug(args ...interface{}) { m.Called(args) }

// Info level logging methods

// Info logs an info message.
func (m *MockLogger) Info(args ...interface{}) { m.Called(args) }

// Warn logs a warning message.
func (m *MockLogger) Warn(args ...interface{}) { m.Called(args) }

// Error logs an error message.
func (m *MockLogger) Error(args ...interface{}) { m.Called(args) }

// Fatal logs a fatal message.
func (m *MockLogger) Fatal(args ...interface{}) { m.Called(args) }

// Panic logs a panic message.
func (m *MockLogger) Panic(args ...interface{}) { m.Called(args) }

// Context methods

// WithFields adds fields to the logger and returns a new logger instance.
func (m *MockLogger) WithFields(fields map[string]interface{}) goai.Logger {
	args := m.Called(fields)
	return args.Get(0).(goai.Logger)
}

// WithContext adds a context to the logger and returns a new logger instance.
func (m *MockLogger) WithContext(ctx context.Context) goai.Logger {
	args := m.Called(ctx)
	return args.Get(0).(goai.Logger)
}

// WithErr adds an error to the logger and returns a new logger instance.
func (m *MockLogger) WithErr(err error) goai.Logger {
	args := m.Called(err)
	return args.Get(0).(goai.Logger)
}
