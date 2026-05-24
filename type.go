// Package config defines the specific parameters of the Contexts and Dev Context
package context

type ContextType = string

const (
	// DevContext indicates that all dependency proxies are in the local machine
	DevContext ContextType = "development"
)
