package config

import (
	"fmt"
	"slices"
	"strings"
)

type CommandDep struct {
	Command    string      `json:"command"`
	Proxies    []DepTarget `json:"proxies,omitempty"`
	Extensions []DepTarget `json:"extensions,omitempty"`
}

type Socket struct {
	Id   string `json:"id"`
	Port int    `json:"port,omitempty"`
}

type Handler struct {
	Type        HandlerType  `json:"type"`
	Category    string       `json:"category"`
	Socket      Socket       `json:"socket"`
	CommandDeps []CommandDep `json:"command-deps,omitempty"`
}

// Service type defined in the config.
//
// Fields
//   - Type is the type of service. For example, ProxyType, IndependentType or ExtensionType
//   - Name of the service
//   - Handlers that are listed in the service
type Service struct {
	Type         Type      `json:"type"`
	Name         string    `json:"name"`
	ModuleUrl    string    `json:"module-url,omitempty"`
	StartCommand string    `json:"start-command,omitempty"`
	Handlers     []Handler `json:"handlers"`
}

// New generates a service configuration.
func New(name string, serviceType Type) *Service {
	return &Service{
		Type:     serviceType,
		Name:     name,
		Handlers: make([]Handler, 0),
	}
}

// ValidateService validates the service metadata and socket bootstrap settings.
func ValidateService(service Service) error {
	if len(service.Name) == 0 {
		return fmt.Errorf("service name is empty")
	}
	if err := ValidateServiceType(service.Type); err != nil {
		return fmt.Errorf("identity.ValidateServiceType: %v", err)
	}

	needsModuleURL := false
	needsStartCommand := false
	for _, h := range service.Handlers {
		if err := ValidateHandlerType(h.Type); err != nil {
			return fmt.Errorf("ValidateHandlerType: %v", err)
		}
		if len(h.Category) == 0 {
			return fmt.Errorf("handler category is empty")
		}
		if len(h.Socket.Id) == 0 {
			return fmt.Errorf("handler '%s' socket id is empty", h.Category)
		}
		if h.Socket.Port < 0 {
			return fmt.Errorf("handler '%s' socket port is negative", h.Category)
		}

		if h.Socket.Port == 0 {
			if strings.HasPrefix(h.Socket.Id, "tmp/") {
				needsStartCommand = true
			} else {
				needsModuleURL = true
			}
		}

		for _, dep := range h.CommandDeps {
			if err := ValidateCommandDep(dep); err != nil {
				return fmt.Errorf("ValidateCommandDep: %v", err)
			}
		}
	}

	if needsModuleURL && len(service.ModuleUrl) == 0 {
		return fmt.Errorf("service('%s') has inproc socket and requires module-url", service.Name)
	}
	if needsStartCommand && len(service.StartCommand) == 0 {
		return fmt.Errorf("service('%s') has tmp socket and requires start-command", service.Name)
	}

	return nil
}

// ValidateTypes validates the parameters of the service.
func (s *Service) ValidateTypes() error {
	if s == nil {
		return fmt.Errorf("service struct is nil")
	}
	return ValidateService(*s)
}

// HandlerByCategory returns the handler config by the handler category.
// If the handler doesn't exist, then it returns an error.
func (s *Service) HandlerByCategory(category string) (Handler, error) {
	if len(category) == 0 {
		return Handler{}, fmt.Errorf("category argument is empty")
	}

	i := slices.IndexFunc(s.Handlers, func(e Handler) bool {
		return e.Category == category
	})
	if i == -1 {
		return Handler{}, fmt.Errorf("handler of '%s' category not found", category)
	}

	return s.Handlers[i], nil
}

// GetHandler returns a handler by its socket id and port.
func (s *Service) GetHandler(id string, port int) (Handler, error) {
	if s == nil {
		return Handler{}, fmt.Errorf("service struct is nil")
	}
	if len(id) == 0 {
		return Handler{}, fmt.Errorf("socket id argument is empty")
	}

	i := slices.IndexFunc(s.Handlers, func(h Handler) bool {
		return h.Socket.Id == id && h.Socket.Port == port
	})
	if i == -1 {
		return Handler{}, fmt.Errorf("handler with socket '%s:%d' not found", id, port)
	}

	return s.Handlers[i], nil
}

// SetHandler adds a new handler.
// If the handler with the same socket id exists, it will over-write that handler.
func (s *Service) SetHandler(handler Handler) {
	if s == nil {
		return
	}

	if len(s.Handlers) == 0 {
		s.Handlers = []Handler{handler}
		return
	}

	i := slices.IndexFunc(s.Handlers, func(h Handler) bool {
		return h.Socket.Id == handler.Socket.Id
	})

	if i == -1 {
		s.Handlers = append(s.Handlers, handler)
		return
	}

	s.Handlers[i] = handler
}

// RemoveHandler removes a handler by its socket.
func (s *Service) RemoveHandler(socket Socket) error {
	if s == nil {
		return fmt.Errorf("service struct is nil")
	}
	if len(socket.Id) == 0 {
		return fmt.Errorf("socket id argument is empty")
	}

	i := slices.IndexFunc(s.Handlers, func(h Handler) bool {
		return h.Socket.Id == socket.Id && h.Socket.Port == socket.Port
	})
	if i == -1 {
		return fmt.Errorf("handler with socket '%s:%d' not found", socket.Id, socket.Port)
	}

	s.Handlers = slices.Delete(s.Handlers, i, i+1)
	return nil
}

// ValidateCommandDep checks that a command dependency declares routing targets.
func ValidateCommandDep(dep CommandDep) error {
	if len(dep.Command) == 0 {
		return fmt.Errorf("command argument is empty")
	}
	if len(dep.Proxies) == 0 && len(dep.Extensions) == 0 {
		return fmt.Errorf("command('%s') must declare proxies or extensions", dep.Command)
	}

	for i, target := range dep.Proxies {
		if err := ValidateDepTarget(target); err != nil {
			return fmt.Errorf("proxies[%d]: %w", i, err)
		}
	}
	for i, target := range dep.Extensions {
		if err := ValidateDepTarget(target); err != nil {
			return fmt.Errorf("extensions[%d]: %w", i, err)
		}
	}

	return nil
}
