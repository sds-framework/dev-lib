package config

import "testing"

func testService() (*Service, Handler, Handler, Handler) {
	handlerOfType := Handler{
		Type:   ReplierType,
		Socket: Socket{Id: "handler_1", Port: 4101},
	}
	handler2OfType := Handler{
		Type:   ReplierType,
		Socket: Socket{Id: "handler_2", Port: 4102},
	}
	handlerOfType2 := Handler{
		Type:   SyncReplierType,
		Socket: Socket{Id: "handler_3", Port: 4103},
	}

	return New("service_id", IndependentType), handlerOfType, handler2OfType, handlerOfType2
}

func TestNewServiceDefaults(t *testing.T) {
	serviceConfig := New("service_id", IndependentType)

	if serviceConfig.StopCommand != DefaultStopCommand {
		t.Fatalf("StopCommand = %q, want %q", serviceConfig.StopCommand, DefaultStopCommand)
	}
	if serviceConfig.StatusCommand != "" {
		t.Fatalf("StatusCommand = %q, want empty", serviceConfig.StatusCommand)
	}
}

func TestServiceValidateTypes(t *testing.T) {
	_, handlerOfType, _, _ := testService()

	invalidHandler := Handler{Type: HandlerType("invalid_handler_type")}

	generatedService := &Service{
		Type:     "the_invalid_type",
		Handlers: []Handler{handlerOfType},
	}

	if err := generatedService.ValidateTypes(); err == nil {
		t.Fatal("ValidateTypes with invalid service type returned nil error")
	}

	generatedService.Type = IndependentType
	if err := generatedService.ValidateTypes(); err != nil {
		t.Fatalf("ValidateTypes valid service: %v", err)
	}

	generatedService.Handlers = []Handler{invalidHandler}
	if err := generatedService.ValidateTypes(); err == nil {
		t.Fatal("ValidateTypes with invalid handler type returned nil error")
	}
}

func TestValidateCommandDep(t *testing.T) {
	if err := ValidateCommandDep(CommandDep{Command: "orphan"}); err == nil {
		t.Fatal("ValidateCommandDep without proxies or extensions returned nil error")
	}

	if err := ValidateCommandDep(CommandDep{
		Command: "call-user-api",
		Proxies: []string{"auth_proxy"},
	}); err != nil {
		t.Fatalf("ValidateCommandDep with proxies: %v", err)
	}

	if err := ValidateCommandDep(CommandDep{
		Command:    "get-user",
		Extensions: []string{"user_service"},
	}); err != nil {
		t.Fatalf("ValidateCommandDep with extensions: %v", err)
	}
}

func TestServiceHandlerByType(t *testing.T) {
	serviceConfig, handlerOfType, handler2OfType, handlerOfType2 := testService()
	nonExistType := HandlerType("not_found_type")
	serviceConfig.Handlers = []Handler{handlerOfType, handler2OfType, handlerOfType2}

	if _, err := serviceConfig.HandlerByType(""); err == nil {
		t.Fatal("HandlerByType with empty type returned nil error")
	}
	if _, err := serviceConfig.HandlersByType(""); err == nil {
		t.Fatal("HandlersByType with empty type returned nil error")
	}

	if _, err := serviceConfig.HandlerByType(nonExistType); err == nil {
		t.Fatal("HandlerByType with missing type returned nil error")
	}
	if _, err := serviceConfig.HandlersByType(nonExistType); err == nil {
		t.Fatal("HandlersByType with missing type returned nil error")
	}

	foundHandler, err := serviceConfig.HandlerByType(ReplierType)
	if err != nil {
		t.Fatalf("HandlerByType replier: %v", err)
	}
	if foundHandler.Socket.Id != handlerOfType.Socket.Id {
		t.Fatalf("handler id = %q, want %q", foundHandler.Socket.Id, handlerOfType.Socket.Id)
	}

	foundHandler, err = serviceConfig.HandlerByType(SyncReplierType)
	if err != nil {
		t.Fatalf("HandlerByType sync replier: %v", err)
	}
	if foundHandler.Socket.Id != handlerOfType2.Socket.Id {
		t.Fatalf("handler id = %q, want %q", foundHandler.Socket.Id, handlerOfType2.Socket.Id)
	}

	foundHandlers, err := serviceConfig.HandlersByType(ReplierType)
	if err != nil {
		t.Fatalf("HandlersByType replier: %v", err)
	}
	if len(foundHandlers) != 2 {
		t.Fatalf("len(foundHandlers) = %d, want 2", len(foundHandlers))
	}
	if foundHandlers[0].Socket.Id != handlerOfType.Socket.Id {
		t.Fatalf("first handler id = %q, want %q", foundHandlers[0].Socket.Id, handlerOfType.Socket.Id)
	}
	if foundHandlers[1].Socket.Id != handler2OfType.Socket.Id {
		t.Fatalf("second handler id = %q, want %q", foundHandlers[1].Socket.Id, handler2OfType.Socket.Id)
	}

	foundHandlers, err = serviceConfig.HandlersByType(SyncReplierType)
	if err != nil {
		t.Fatalf("HandlersByType sync replier: %v", err)
	}
	if len(foundHandlers) != 1 {
		t.Fatalf("len(foundHandlers) = %d, want 1", len(foundHandlers))
	}
	if foundHandlers[0].Socket.Id != handlerOfType2.Socket.Id {
		t.Fatalf("handler id = %q, want %q", foundHandlers[0].Socket.Id, handlerOfType2.Socket.Id)
	}
}

func TestServiceGetHandler(t *testing.T) {
	serviceConfig, handlerOfType, _, handlerOfType2 := testService()
	serviceConfig.Handlers = []Handler{
		handlerOfType,
		{
			Type:   PairType,
			Socket: Socket{Id: handlerOfType.Socket.Id, Port: 9999},
		},
		handlerOfType2,
	}

	if _, err := serviceConfig.GetHandler("", handlerOfType.Socket.Port); err == nil {
		t.Fatal("GetHandler with empty id returned nil error")
	}
	if _, err := serviceConfig.GetHandler(handlerOfType.Socket.Id, 1234); err == nil {
		t.Fatal("GetHandler with missing socket returned nil error")
	}

	foundHandler, err := serviceConfig.GetHandler(handlerOfType.Socket.Id, handlerOfType.Socket.Port)
	if err != nil {
		t.Fatalf("GetHandler: %v", err)
	}
	if foundHandler.Type != handlerOfType.Type {
		t.Fatalf("handler type = %q, want %q", foundHandler.Type, handlerOfType.Type)
	}
}

func TestServiceSetHandler(t *testing.T) {
	serviceConfig, handlerOfType, _, handlerOfType2 := testService()

	if len(serviceConfig.Handlers) != 0 {
		t.Fatalf("initial len(Handlers) = %d, want 0", len(serviceConfig.Handlers))
	}

	var nilService *Service
	nilService.SetHandler(handlerOfType)

	serviceConfig.SetHandler(handlerOfType)
	if len(serviceConfig.Handlers) != 1 {
		t.Fatalf("len(Handlers) = %d, want 1", len(serviceConfig.Handlers))
	}
	if serviceConfig.Handlers[0].Type != ReplierType {
		t.Fatalf("handler type = %q, want %q", serviceConfig.Handlers[0].Type, ReplierType)
	}

	serviceConfig.SetHandler(handlerOfType2)
	if len(serviceConfig.Handlers) != 2 {
		t.Fatalf("len(Handlers) = %d, want 2", len(serviceConfig.Handlers))
	}
	if serviceConfig.Handlers[0].Type != ReplierType {
		t.Fatalf("first handler type = %q, want %q", serviceConfig.Handlers[0].Type, ReplierType)
	}
	if serviceConfig.Handlers[1].Type != SyncReplierType {
		t.Fatalf("second handler type = %q, want %q", serviceConfig.Handlers[1].Type, SyncReplierType)
	}

	updatedHandler := Handler{
		Type:   PairType,
		Socket: Socket{Id: handlerOfType.Socket.Id},
	}
	serviceConfig.SetHandler(updatedHandler)
	if len(serviceConfig.Handlers) != 2 {
		t.Fatalf("len(Handlers) after update = %d, want 2", len(serviceConfig.Handlers))
	}
	if serviceConfig.Handlers[0].Type != PairType {
		t.Fatalf("first handler type = %q, want %q", serviceConfig.Handlers[0].Type, PairType)
	}
	if serviceConfig.Handlers[1].Type != SyncReplierType {
		t.Fatalf("second handler type = %q, want %q", serviceConfig.Handlers[1].Type, SyncReplierType)
	}
}

func TestServiceRemoveHandler(t *testing.T) {
	serviceConfig, handlerOfType, handler2OfType, handlerOfType2 := testService()
	serviceConfig.Handlers = []Handler{handlerOfType, handler2OfType, handlerOfType2}

	if err := serviceConfig.RemoveHandler(Socket{}); err == nil {
		t.Fatal("RemoveHandler with empty socket returned nil error")
	}
	if err := serviceConfig.RemoveHandler(Socket{Id: handlerOfType.Socket.Id, Port: 9999}); err == nil {
		t.Fatal("RemoveHandler with missing socket returned nil error")
	}

	if err := serviceConfig.RemoveHandler(handler2OfType.Socket); err != nil {
		t.Fatalf("RemoveHandler: %v", err)
	}
	if len(serviceConfig.Handlers) != 2 {
		t.Fatalf("len(Handlers) = %d, want 2", len(serviceConfig.Handlers))
	}
	if serviceConfig.Handlers[0].Socket.Id != handlerOfType.Socket.Id {
		t.Fatalf("first handler id = %q, want %q", serviceConfig.Handlers[0].Socket.Id, handlerOfType.Socket.Id)
	}
	if serviceConfig.Handlers[1].Socket.Id != handlerOfType2.Socket.Id {
		t.Fatalf("second handler id = %q, want %q", serviceConfig.Handlers[1].Socket.Id, handlerOfType2.Socket.Id)
	}
}
