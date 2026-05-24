// Package runtime contains the dependency runtime for the dev context.
package runtime

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sds-framework/client-lib"
	clientConfig "github.com/sds-framework/client-lib/config"
	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/datatype-lib/data_type/key_value"
	"github.com/sds-framework/datatype-lib/message"
	handlerConfig "github.com/sds-framework/handler-lib/config"
	"github.com/sds-framework/log-lib"
)

// DefaultTimeout is the default time to wait before considering the message is not delivered.
// Runtime.IsServiceRunning method uses this value before considering the socket as not running.
const DefaultTimeout = time.Second

type Process struct {
	config *config.Service
	id     string
	cmd    *exec.Cmd
	done   chan error // signalizes when the service finished
}

// Runtime runs, stops, and checks dependency services.
type Runtime struct {
	config           *config.SdsService
	sameServices     map[string]int
	runningProcesses map[string]*Process
	timeout          time.Duration
}

// AddService registers a service in the runtime configuration.
func (rt *Runtime) AddService(service config.Service) error {
	if rt == nil || rt.config == nil {
		return fmt.Errorf("nil config")
	}
	if len(service.Name) == 0 {
		return fmt.Errorf("service name is empty")
	}

	if err := rt.config.SetService(service); err != nil {
		return fmt.Errorf("rt.config.SetService: %w", err)
	}

	return rt.config.Save()
}

// RemoveService removes a service from the runtime configuration.
func (rt *Runtime) RemoveService(serviceName string) error {
	if rt == nil || rt.config == nil {
		return fmt.Errorf("nil config")
	}
	if len(serviceName) == 0 {
		return fmt.Errorf("service name is empty")
	}

	running, err := rt.serviceRunning(serviceName)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("service('%s') is running, please stop it first", serviceName)
	}

	if err := rt.config.RemoveService(serviceName); err != nil {
		return err
	}

	if err := rt.config.Save(); err != nil {
		return fmt.Errorf("rt.config.Save: %w", err)
	}

	delete(rt.sameServices, serviceName)
	return nil
}

// New creates a dependency runtime in the Dev context.
func New(cfg *config.SdsService) *Runtime {
	return &Runtime{
		config:           cfg,
		sameServices:     make(map[string]int),
		runningProcesses: make(map[string]*Process, 0),
		timeout:          DefaultTimeout,
	}
}

func (process *Process) copy() *Process {
	return &Process{
		config: process.config,
		id:     process.id,
		done:   make(chan error, 1),
	}
}

// StopService stops the dependency service.
func (rt *Runtime) StopService(c *clientConfig.Client) error {
	// Make sure it's running
	running, err := rt.IsServiceRunning(c)
	if err != nil {
		return fmt.Errorf("runtime.IsServiceRunning(client='%v'): %w", *c, err)
	}
	if !running {
		return nil
	}

	sock, err := client.New(c)
	if err != nil {
		return fmt.Errorf("zmq.NewSocket: %w", err)
	}

	closeRequest := &message.Request{
		Command:    "close",
		Parameters: key_value.New(),
	}

	sock.Timeout(rt.timeout).Attempt(1)

	_, err = sock.Request(closeRequest)
	if err == nil {
		return fmt.Errorf("socket.Request('close'): must exist with error since service closed before replying back")
	}

	running, err = rt.IsServiceRunning(c)
	if err != nil {
		return fmt.Errorf("socket.Request('close'): runtime.IsServiceRunning(client='%v'): %w", *c, err)
	}

	if running {
		return fmt.Errorf("runtime is running even after closing")
	}

	err = sock.Close()
	if err != nil {
		return fmt.Errorf("socket.Close: %w", err)
	}

	return nil
}

// IsServiceRunning checks whether the given client running or not.
// If the service is running on another process or on another node,
// then that service should expose the port.
func (rt *Runtime) IsServiceRunning(c *clientConfig.Client) (bool, error) {
	c.UrlFunc(clientConfig.Url)

	sock, err := client.New(c)
	if err != nil {
		return false, fmt.Errorf("client.New: %w", err)
	}
	sock.Attempt(1).Timeout(rt.timeout)

	req := &message.Request{
		Command:    "heartbeat",
		Parameters: key_value.New(),
	}

	_, err = sock.Request(req)
	if err != nil {
		return false, nil
	}

	closeErr := sock.Close()
	if closeErr != nil {
		return false, fmt.Errorf("socket.Close: %w", err)
	}

	return true, nil
}

// OnStop returns a signal through the channel when the process spawned by the Runtime stops.
// If the process is not existing, then it will simply return error.
func (rt *Runtime) OnStop(id string) chan error {
	process, ok := rt.runningProcesses[id]
	if !ok {
		return nil
	}

	if process.cmd == nil {
		return nil
	}

	return process.done
}

// GenerateId returns the next runtime id for a service name.
func (rt *Runtime) GenerateId(serviceName string) (string, error) {
	if rt == nil {
		return "", fmt.Errorf("nil runtime")
	}
	if len(serviceName) == 0 {
		return "", fmt.Errorf("service name is empty")
	}
	if rt.sameServices == nil {
		rt.sameServices = make(map[string]int)
	}

	count := rt.sameServices[serviceName]
	for {
		count++
		id := fmt.Sprintf("%s%d", serviceName, count)
		if _, exists := rt.runningProcesses[id]; !exists {
			rt.sameServices[serviceName]++
			return id, nil
		}
	}
}

func (rt *Runtime) refreshServiceCount(serviceName string) {
	count := 0
	for _, process := range rt.runningProcesses {
		if process != nil && process.config != nil && process.config.Name == serviceName {
			count++
		}
	}
	if count == 0 {
		delete(rt.sameServices, serviceName)
		return
	}
	rt.sameServices[serviceName] = count
}

func managerHandler(service *config.Service) (config.Handler, error) {
	for _, handler := range service.Handlers {
		if strings.Contains(handler.Socket.Id, "manager") {
			return handler, nil
		}
	}

	return config.Handler{}, fmt.Errorf("manager handler not found for service('%s')", service.Name)
}

func (rt *Runtime) serviceRunning(serviceName string) (bool, error) {
	service, err := rt.serviceConfig(serviceName)
	if err != nil {
		return false, err
	}

	handler, err := managerHandler(service)
	if err != nil {
		return false, err
	}

	client := clientConfig.New(
		service.Name,
		handler.Socket.Id,
		uint64(handler.Socket.Port),
		handlerConfig.SocketType(handlerConfig.HandlerType(handler.Type)),
	)

	return rt.IsServiceRunning(client)
}

func (rt *Runtime) serviceConfig(serviceName string) (*config.Service, error) {
	if rt == nil || rt.config == nil {
		return nil, fmt.Errorf("nil config")
	}
	if len(serviceName) == 0 {
		return nil, fmt.Errorf("service name is empty")
	}

	for i := range rt.config.Services {
		if rt.config.Services[i].Name == serviceName {
			return &rt.config.Services[i], nil
		}
	}

	return nil, fmt.Errorf("service('%s') not found", serviceName)
}

// StartService runs the service start command.
// If it fails to run, then it will return an error.
//
// Note that, services can crash during the initialization.
// In that case, you should use Runtime.OnStop method.
//
// If a parent is given, it's passed as ParentFlag.
// Todo, move all Flags from service-lib to config-lig.
// Todo, use the ParentFlag from the config lig
func (rt *Runtime) StartService(serviceName string, optionalParent ...*clientConfig.Client) (string, error) {
	if rt == nil {
		return "", fmt.Errorf("nil runtime")
	}
	serviceConfig, err := rt.serviceConfig(serviceName)
	if err != nil {
		return "", err
	}
	process := &Process{config: serviceConfig}

	if len(optionalParent) > 1 {
		return "", fmt.Errorf("too many optional parameters, either no parameter or 1 parameter required")
	}
	if len(optionalParent) == 1 && optionalParent[0] == nil {
		return "", fmt.Errorf("nil parent")
	}

	if len(process.config.StartCommand) == 0 {
		return "", fmt.Errorf("no start command")
	}

	id, err := rt.GenerateId(process.config.Name)
	if err != nil {
		return "", fmt.Errorf("rt.GenerateId('%s'): %w", process.config.Name, err)
	}
	process.id = id

	idFlag := fmt.Sprintf("--id=%s", id)

	args := []string{idFlag}

	if len(optionalParent) == 1 {
		parentKv, err := key_value.NewFromInterface(optionalParent[0])
		if err != nil {
			rt.refreshServiceCount(process.config.Name)
			return "", fmt.Errorf("optionalParent: key_value.NewFromInterface(parent='%v'): %w", optionalParent[0], err)
		}
		parentFlag := fmt.Sprintf("--parent=%s", parentKv.String())
		args = append(args, parentFlag)
	}

	commandArgs := strings.Fields(process.config.StartCommand)
	if len(commandArgs) == 0 {
		rt.refreshServiceCount(process.config.Name)
		return "", fmt.Errorf("no start command")
	}

	instance := process.copy()

	rt.runningProcesses[id] = instance

	logger, err := log.New(id, false)
	if err != nil {
		delete(rt.runningProcesses, id)
		rt.refreshServiceCount(process.config.Name)
		return "", fmt.Errorf("log.New('%s'): %w", id, err)
	}
	errLogger, err := log.New(id+"Err", false)
	if err != nil {
		delete(rt.runningProcesses, id)
		rt.refreshServiceCount(process.config.Name)
		return "", fmt.Errorf("log.New('%sErr'): %w", id, err)
	}

	cmd := exec.Command(commandArgs[0], append(commandArgs[1:], args...)...)
	cmd.Stdout = logger
	cmd.Stderr = errLogger
	err = cmd.Start()
	if err != nil {
		delete(rt.runningProcesses, id)
		rt.refreshServiceCount(process.config.Name)
		return "", fmt.Errorf("cmd.Start: %w", err)
	}

	instance.cmd = cmd
	rt.wait(id)

	return id, nil
}

// The wait is invoked if the spawned dependency stops.
// The dependencies are running asynchronously.
// In order to call this function, you must use the Runtime.StopService() method.
// If the Close signal was sent to the spawned child, then
// this method will be called automatically by the operating system.
func (rt *Runtime) wait(id string) {
	go func() {
		process := rt.runningProcesses[id]
		err := process.cmd.Wait() // it can return an error
		process.done <- err
		delete(rt.runningProcesses, id)
		rt.refreshServiceCount(process.config.Name)
	}()
}
