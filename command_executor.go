package mcptools

import (
	"context"
	"os/exec"
)

// CommandExecutor interface for executing commands
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error)
}

// RealCommandExecutor implements CommandExecutor for real command execution
type RealCommandExecutor struct{}

func (e *RealCommandExecutor) ExecuteCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}
