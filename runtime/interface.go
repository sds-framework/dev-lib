package runtime

import (
	clientConfig "github.com/sds-framework/client-lib/config"
	config "github.com/sds-framework/config-lib"
)

// Interface is implemented by the dependency runtime.
//
// It doesn't have the `Stop` command.
// Because, stopping must be done by the remote call from other services.
type Interface interface {
	// AddService registers a service in the runtime configuration.
	AddService(service config.Service) error

	// RemoveService removes a service from the runtime configuration.
	RemoveService(serviceName string) error

	// StartService starts the dependency service with the given parent.
	StartService(serviceName string, optionalParent ...*clientConfig.Client) (string, error)

	// IsServiceRunning checks is the service running or not.
	IsServiceRunning(*clientConfig.Client) (bool, error)

	// StopService stops the given dependency service.
	StopService(c *clientConfig.Client) error
}
