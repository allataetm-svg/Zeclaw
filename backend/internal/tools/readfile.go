package tools

import (
	"context"
	"fmt"
	"os"
)

const maxFileBytes = 50 * 1024 // 50 KB

// ReadFileTool reads files from the local filesystem (scoped to the
// configured workspace directory when a relative path is given).
type ReadFileTool struct {
	workspaceDir string
}

// NewReadFileTool creates a ReadFileTool whose relative paths are resolved
// against workspaceDir.
func NewReadFileTool(workspaceDir string) *ReadFileTool {
	return &ReadFileTool{workspaceDir: workspaceDir}
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a file" }

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

// Execute reads the file at the given path and returns its contents,
// truncating at 50 KB if necessary.
func (t *ReadFileTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Respect context cancellation before doing I/O.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	content := string(data)
	if len(content) > maxFileBytes {
		content = content[:maxFileBytes] + "\n[file truncated]"
	}

	return content, nil
}
