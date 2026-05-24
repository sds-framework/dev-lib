// Package runtime contains the dependency runtime for the dev context.
package runtime

import (
	"fmt"

	clientConfig "github.com/sds-framework/client-lib/config"
	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/datatype-lib/data_type/key_value"
	"github.com/sds-framework/datatype-lib/message"
	"github.com/sds-framework/handler-lib/base"
	handlerConfig "github.com/sds-framework/handler-lib/config"
	"github.com/sds-framework/handler-lib/replier"
	"github.com/sds-framework/log-lib"
)

const (
	RuntimeHandlerCategory = "dep_handler" // handler category
	RuntimeSocketType      = handlerConfig.ReplierType
	IsServiceRunning       = "is-service-running"
	StartService           = "start-service"
	StopService            = "stop-service"
	AddService             = "add-service"
	RemoveService          = "remove-service"
)

// Handler acts as the router from other app processes to the runtime.
type Handler struct {
	handler base.Interface // Receive commands
	runtime Interface      // Route to the functions from runtime
}

// HandlerConfig returns the handler configuration for the runtime socket.
func HandlerConfig(runtimeSocket config.Socket) *handlerConfig.Handler {
	return handlerConfig.NewHandler(
		RuntimeSocketType,
		runtimeSocket.Id,
		RuntimeHandlerCategory,
		uint64(runtimeSocket.Port),
	)
}

// NewHandler returns a dependency runtime handler.
func NewHandler(cfg *config.SdsService, runtimeSocket config.Socket) (*Handler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	handler := replier.New()

	logger, err := log.New("dep_runtime", true)
	if err != nil {
		return nil, fmt.Errorf("log.New('dep-handler'): %w", err)
	}

	handler.SetConfig(HandlerConfig(runtimeSocket))
	err = handler.SetLogger(logger)
	if err != nil {
		return nil, fmt.Errorf("handler.SetLogger: %w", err)
	}

	return &Handler{
		runtime: New(cfg),
		handler: handler,
	}, nil
}

// onIsServiceRunning checks whether the dependency is running or not.
// Requires:
//   - 'dep' of the clientConfig.Client.
//
// Returns 'running' boolean result.
func (h *Handler) onIsServiceRunning(req message.RequestInterface) message.ReplyInterface {
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
func (h *Handler) onStartService(req message.RequestInterface) message.ReplyInterface {
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
func (h *Handler) onAddService(req message.RequestInterface) message.ReplyInterface {
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
func (h *Handler) onRemoveService(req message.RequestInterface) message.ReplyInterface {
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
func (h *Handler) onStopService(req message.RequestInterface) message.ReplyInterface {
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

// Start starts the dependency handler with the available operations.
func (h *Handler) Start() error {
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
