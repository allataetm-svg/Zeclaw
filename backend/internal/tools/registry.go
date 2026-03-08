package tools

import (
	"context"
	"fmt"
)

// Registry holds all registered tools and provides look-up and execution
// helpers used by the agent loop.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds tool to the registry, keyed on tool.Name().
// If a tool with the same name was already registered it is replaced.
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools in an unspecified order.
func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// ListForAgent returns OpenAI-compatible tool definition objects for all
// tools whose names are in allowedTools.  If allowedTools is empty every
// registered tool is included.
func (r *Registry) ListForAgent(allowedTools []string) []map[string]interface{} {
	allowed := make(map[string]bool, len(allowedTools))
	for _, name := range allowedTools {
		allowed[name] = true
	}

	defs := make([]map[string]interface{}, 0, len(r.tools))
	for name, tool := range r.tools {
		if len(allowedTools) > 0 && !allowed[name] {
			continue
		}
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}
	return defs
}

// Execute looks up the named tool and runs it with the provided arguments.
func (r *Registry) Execute(ctx context.Context, name string, input map[string]interface{}) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(ctx, input)
}
