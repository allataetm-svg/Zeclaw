package tools

import "context"

// Tool is the interface that all agent tools must implement.
type Tool interface {
	// Name returns the unique identifier used when the LLM calls the tool.
	Name() string

	// Description is a human-readable summary that helps the LLM decide when
	// to use this tool.
	Description() string

	// Parameters returns a JSON-Schema-compatible object description of the
	// tool's input arguments (used to build the tools array sent to the LLM).
	Parameters() map[string]interface{}

	// Execute runs the tool with the provided arguments and returns its output
	// or an error.  Implementations should respect context cancellation.
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}
