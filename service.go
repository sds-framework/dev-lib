package config

import (
	"fmt"
	"slices"
)

const DefaultStopCommand = "SIGTERM"

type CommandDep struct {
	Command    string   `json:"command"`
	Proxies    []string `json:"proxies,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
}

type Socket struct {
	Id   string `json:"id"`
	Port int    `json:"port"`
}

type Handler struct {
	Type        HandlerType  `json:"type"`
	Socket      Socket       `json:"socket"`
	CommandDeps []CommandDep `json:"command-deps,omitempty"`
}

// Service type defined in the config.
//
// Fields
//   - Type is the type of service. For example, ProxyType, IndependentType or ExtensionType
//   - Name of the service
//   - StartCommand is the command used to start the service
//   - StopCommand is the command used to stop the service. Defaults to SIGTERM
//   - StatusCommand is an optional command used to check service status
//   - Handlers that are listed in the service
type Service struct {
	Type          Type      `json:"type"`
	Name          string    `json:"name"`
	StartCommand  string    `json:"start-command"`
	StopCommand   string    `json:"stop-command"`
	StatusCommand string    `json:"status-command,omitempty"`
	Handlers      []Handler `json:"handlers"`
}

// New generates a service configuration.
func New(name string, serviceType Type) *Service {
	return &Service{
		Type:        serviceType,
		Name:        name,
		StopCommand: DefaultStopCommand,
		Handlers:    make([]Handler, 0),
	}
}

func (s *Service) setDefaults() {
	if len(s.StopCommand) == 0 {
		s.StopCommand = DefaultStopCommand
	}
}

// ValidateTypes the parameters of the service
func (s *Service) ValidateTypes() error {
	if err := ValidateServiceType(s.Type); err != nil {
		return fmt.Errorf("identity.ValidateServiceType: %v", err)
	}

	for _, c := range s.Handlers {
		if err := ValidateHandlerType(c.Type); err != nil {
			return fmt.Errorf("ValidateHandlerType: %v", err)
		}

		for _, dep := range c.CommandDeps {
			if err := ValidateCommandDep(dep); err != nil {
				return fmt.Errorf("ValidateCommandDep: %v", err)
			}
		}
	}

	return nil
}

// HandlerByType returns the handler config by the handler type.
// If the handler doesn't exist, then it returns an error.
func (s *Service) HandlerByType(handlerType HandlerType) (Handler, error) {
	if len(handlerType) == 0 {
		return Handler{}, fmt.Errorf("handlerType argument is empty")
	}

	i := slices.IndexFunc(s.Handlers, func(e Handler) bool {
		return e.Type == handlerType
	})
	if i == -1 {
		return Handler{}, fmt.Errorf("handler of '%s' type not found", handlerType)
	}

	return s.Handlers[i], nil
}

// HandlersByType returns the multiple handlers of the given type.
// If the handlers don't exist, then it returns an error
func (s *Service) HandlersByType(handlerType HandlerType) ([]Handler, error) {
	if len(handlerType) == 0 {
		return nil, fmt.Errorf("handlerType argument is empty")
	}

	handlers := make([]Handler, 0)
	i := 0

	for _, c := range s.Handlers {
		if c.Type == handlerType {
			handlers = slices.Grow(handlers, 1)
			handlers = slices.Insert(handlers, i, c)
			i++
		}
	}

	if len(handlers) == 0 {
		return nil, fmt.Errorf("no '%s' handlers config", handlerType)
	}
	return handlers, nil
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

	return nil
}
