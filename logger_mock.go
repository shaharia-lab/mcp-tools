package mcptools

import (
	"context"
	"fmt"
	"github.com/shaharia-lab/goai/observability"
	"sync"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	mu sync.Mutex
	// Store logs by level
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
	fatalLogs []string
	panicLogs []string
	// Store formatted logs by level
	debugfLogs []string
	infofLogs  []string
	warnfLogs  []string
	errorfLogs []string
	fatalfLogs []string
	panicfLogs []string
	// Store fields for WithFields calls
	fields map[string]interface{}
	// Store context for WithContext calls
	ctx context.Context
	// Store error for WithErr calls
	err error
}

// NewMockLogger creates a new MockLogger instance
func NewMockLogger() *MockLogger {
	return &MockLogger{
		fields: make(map[string]interface{}),
	}
}

// Helper method to format args
func formatArgs(args ...interface{}) string {
	return fmt.Sprint(args...)
}

// Implement formatted logging methods
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugfLogs = append(m.debugfLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infofLogs = append(m.infofLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnfLogs = append(m.warnfLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorfLogs = append(m.errorfLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fatalfLogs = append(m.fatalfLogs, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Panicf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panicfLogs = append(m.panicfLogs, fmt.Sprintf(format, args...))
}

// Implement regular logging methods
func (m *MockLogger) Debug(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugLogs = append(m.debugLogs, formatArgs(args...))
}

func (m *MockLogger) Info(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoLogs = append(m.infoLogs, formatArgs(args...))
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnLogs = append(m.warnLogs, formatArgs(args...))
}

func (m *MockLogger) Error(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorLogs = append(m.errorLogs, formatArgs(args...))
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fatalLogs = append(m.fatalLogs, formatArgs(args...))
}

func (m *MockLogger) Panic(args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panicLogs = append(m.panicLogs, formatArgs(args...))
}

// Implement context methods
func (m *MockLogger) WithFields(fields map[string]interface{}) observability.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	newLogger := NewMockLogger()
	for k, v := range m.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (m *MockLogger) WithContext(ctx context.Context) observability.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	newLogger := NewMockLogger()
	newLogger.ctx = ctx
	newLogger.fields = m.fields
	return newLogger
}

func (m *MockLogger) WithErr(err error) observability.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	newLogger := NewMockLogger()
	newLogger.err = err
	newLogger.fields = m.fields
	return newLogger
}

func (m *MockLogger) GetDebugLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.debugLogs...)
}

func (m *MockLogger) GetInfoLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.infoLogs...)
}

func (m *MockLogger) GetWarnLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.warnLogs...)
}

func (m *MockLogger) GetErrorLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.errorLogs...)
}

func (m *MockLogger) GetFields() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	fields := make(map[string]interface{})
	for k, v := range m.fields {
		fields[k] = v
	}
	return fields
}

func (m *MockLogger) GetContext() context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ctx
}

func (m *MockLogger) GetError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}
