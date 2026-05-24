// Package dep_handler creates a thread that manages the dependencies
package dep_handler

import (
	"fmt"

	clientConfig "github.com/sds-framework/client-lib/config"
	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/datatype-lib/data_type/key_value"
	"github.com/sds-framework/datatype-lib/message"
	"github.com/sds-framework/dev-lib/runtime"
	"github.com/sds-framework/handler-lib/base"
	handlerConfig "github.com/sds-framework/handler-lib/config"
	"github.com/sds-framework/handler-lib/replier"
	"github.com/sds-framework/log-lib"
)

const (
	Category         = "dep_handler" // handler category
	IsServiceRunning = "is-service-running"
	StartService     = "start-service"
	StopService      = "stop-service"
	AddService       = "add-service"
	RemoveService    = "remove-service"
)

// Handler acts as the router from other app processes to the runtime.
type DepHandler struct {
	handler base.Interface    // Receive commands
	runtime runtime.Interface // Route to the functions from runtime
}

// ServiceConfig returns the socket configuration of the handler
func ServiceConfig() *handlerConfig.Handler {
	return handlerConfig.NewInternalHandler(handlerConfig.ReplierType, Category, Category)
}

// New dep handler returned
func New(cfg *config.SdsService) (*DepHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	handler := replier.New()

	logger, err := log.New("dep_runtime", true)
	if err != nil {
		return nil, fmt.Errorf("log.New('dep-handler'): %w", err)
	}

	handler.SetConfig(ServiceConfig())
	err = handler.SetLogger(logger)
	if err != nil {
		return nil, fmt.Errorf("handler.SetLogger: %w", err)
	}

	return &DepHandler{
		runtime: runtime.New(cfg),
		handler: handler,
	}, nil
}

// onIsServiceRunning checks whether the dependency is running or not.
// Requires:
//   - 'dep' of the clientConfig.Client.
//
// Returns 'running' boolean result
func (h *DepHandler) onIsServiceRunning(req message.RequestInterface) message.ReplyInterface {
	kv, err := req.RouteParameters().NestedValue("dep")
	if err != nil {
		return req.Fail(fmt.Sprintf("req.Parameters.GetKeyValue('dep'): %v", err))
	}

	var c clientConfig.Client
	err = kv.Interface(&c)
	if err != nil {
		return req.Fail(fmt.Sprintf("kv.Interface: %v", err))
	}

	c.UrlFunc(clientConfig.Url)

	running, err := h.runtime.IsServiceRunning(&c)
	if err != nil {
		return req.Fail(fmt.Sprintf("h.runtime.IsServiceRunning: %v", err))
	}

	params := key_value.New().Set("running", running)
	return req.Ok(params)
}

// onStartService starts the dependency service.
// Requires:
//   - 'service' string parameter,
//   - 'parent' of the clientConfig.Client type.
//
// Returns nothing.
// todo make it publish the result through publisher, so user won't wait for the result.
func (h *DepHandler) onStartService(req message.RequestInterface) message.ReplyInterface {
	kv, err := req.RouteParameters().NestedValue("parent")
	if err != nil {
		return req.Fail(fmt.Sprintf("req.Parameters.GetKeyValue('parent'): %v", err))
	}

	var parent clientConfig.Client
	err = kv.Interface(&parent)
	if err != nil {
		return req.Fail(fmt.Sprintf("kv.Interface: %v", err))
	}

	parent.UrlFunc(clientConfig.Url)

	serviceName, err := req.RouteParameters().StringValue("service")
	if err != nil {
		serviceName, err = req.RouteParameters().StringValue("url")
		if err != nil {
			return req.Fail(fmt.Sprintf("req.Parameters.GetString('service'): %v", err))
		}
	}

	id, err := h.runtime.StartService(serviceName, &parent)
	if err != nil {
		return req.Fail(fmt.Sprintf("h.runtime.StartService(service: '%s'): %v", serviceName, err))
	}

	return req.Ok(key_value.New().Set("id", id))
}

// onAddService registers a service in the runtime configuration.
// Requires 'service' of the config.Service type.
func (h *DepHandler) onAddService(req message.RequestInterface) message.ReplyInterface {
	kv, err := req.RouteParameters().NestedValue("service")
	if err != nil {
		return req.Fail(fmt.Sprintf("req.Parameters.GetKeyValue('service'): %v", err))
	}

	var service config.Service
	err = kv.Interface(&service)
	if err != nil {
		return req.Fail(fmt.Sprintf("kv.Interface: %v", err))
	}

	err = h.runtime.AddService(service)
	if err != nil {
		return req.Fail(fmt.Sprintf("h.runtime.AddService('%s'): %v", service.Name, err))
	}

	return req.Ok(key_value.New())
}

// onRemoveService removes a service from the runtime configuration.
// Requires 'service' string parameter with the service name.
func (h *DepHandler) onRemoveService(req message.RequestInterface) message.ReplyInterface {
	serviceName, err := req.RouteParameters().StringValue("service")
	if err != nil {
		return req.Fail(fmt.Sprintf("req.Parameters.GetString('service'): %v", err))
	}

	err = h.runtime.RemoveService(serviceName)
	if err != nil {
		return req.Fail(fmt.Sprintf("h.runtime.RemoveService('%s'): %v", serviceName, err))
	}

	return req.Ok(key_value.New())
}

// onStopService stops the dependency.
// Requires 'dep' of the clientConfig.Client type.
// Returns nothing.
//
// Todo make it publish the result through publisher, so user won't wait for the result.
func (h *DepHandler) onStopService(req message.RequestInterface) message.ReplyInterface {
	kv, err := req.RouteParameters().NestedValue("dep")
	if err != nil {
		return req.Fail(fmt.Sprintf("req.Parameters.GetKeyValue('dep'): %v", err))
	}

	var c clientConfig.Client
	err = kv.Interface(&c)
	if err != nil {
		return req.Fail(fmt.Sprintf("kv.Interface: %v", err))
	}

	c.UrlFunc(clientConfig.Url)

	err = h.runtime.StopService(&c)
	if err != nil {
		return req.Fail(fmt.Sprintf("h.runtime.StopService: %v", err))
	}

	return req.Ok(key_value.New())
}

// Start the dependency handler with the available operations.
func (h *DepHandler) Start() error {
	if err := h.handler.Route(IsServiceRunning, h.onIsServiceRunning); err != nil {
		return fmt.Errorf("h.handler.Route('%s'): %v", IsServiceRunning, err)
	}
	if err := h.handler.Route(StartService, h.onStartService); err != nil {
		return fmt.Errorf("h.handler.Route('%s'): %v", StartService, err)
	}
	if err := h.handler.Route(StopService, h.onStopService); err != nil {
		return fmt.Errorf("h.handler.Route('%s'): %v", StopService, err)
	}
	if err := h.handler.Route(AddService, h.onAddService); err != nil {
		return fmt.Errorf("h.handler.Route('%s'): %v", AddService, err)
	}
	if err := h.handler.Route(RemoveService, h.onRemoveService); err != nil {
		return fmt.Errorf("h.handler.Route('%s'): %v", RemoveService, err)
	}

	return h.handler.Start()
}
