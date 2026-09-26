package claude

// Small helpers for writing JSON-schema tool inputs.

type Schema = map[string]any

func Str() Schema             { return Schema{"type": "string"} }
func StrD(desc string) Schema { return Schema{"type": "string", "description": desc} }
func Bool() Schema            { return Schema{"type": "boolean"} }
func Arr(items any) Schema    { return Schema{"type": "array", "items": items} }
func ArrD(items any, d string) Schema {
	return Schema{"type": "array", "items": items, "description": d}
}
func Obj(props Schema, required ...string) Schema {
	s := Schema{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}
