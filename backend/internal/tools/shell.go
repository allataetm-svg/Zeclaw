package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const (
	shellTimeout   = 60 * time.Second
	maxOutputBytes = 10 * 1024 // 10 KB
)

// ShellTool executes arbitrary shell commands on the host device.
type ShellTool struct{}

// NewShellTool returns a ready-to-use ShellTool.
func NewShellTool() *ShellTool { return &ShellTool{} }

func (t *ShellTool) Name() string        { return "shell" }
func (t *ShellTool) Description() string { return "Execute a shell command on the device" }

func (t *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the shell command inside a 60-second timeout context and
// returns the combined stdout/stderr (truncated to 10 KB).
func (t *ShellTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("command is required")
	}

	execCtx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", command)
	// Use a new process group so we can kill the whole tree on cancel.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err := cmd.Run()

	output := outBuf.String()
	if len(output) > maxOutputBytes {
		output = output[:maxOutputBytes] + "\n[output truncated]"
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("command timed out after %v", shellTimeout)
		}
		// A non-zero exit code is informational, not a tool failure; return
		// the output so the LLM can reason about it.
		return output, nil
	}

	return output, nil
}
