package voiceagent

import "encoding/json"

func jsonUnmarshalLenient(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}

type Tool struct {
	Name        string
	Description string

	Parameters map[string]any

	Handler func(args map[string]any) (string, error)
}

func toolSchema(tools []Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		fn := map[string]any{
			"name":        t.Name,
			"description": t.Description,
		}
		if t.Parameters != nil {
			fn["parameters"] = t.Parameters
		}
		out = append(out, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	return out
}

func findTool(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

func parseArgs(s string) map[string]any {
	out := map[string]any{}
	if s == "" {
		return out
	}
	_ = jsonUnmarshalLenient(s, &out)
	return out
}
