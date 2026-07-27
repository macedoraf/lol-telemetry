// Package hooks contains trigger conditions that decide when to invoke the Judge.
package hooks

// Configurable is an optional interface for hooks that support runtime parameter changes.
type Configurable interface {
	Configure(params map[string]any) error
	Spec() map[string]ParamSpec
}

// ParamSpec describes a single configurable parameter.
type ParamSpec struct {
	Type    string  `json:"type"`
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
}
