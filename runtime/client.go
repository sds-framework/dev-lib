package runtime

import (
	"fmt"
	"time"

	"github.com/sds-framework/client-lib"
	clientConfig "github.com/sds-framework/client-lib/config"
	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/datatype-lib/data_type/key_value"
	"github.com/sds-framework/datatype-lib/message"
	handlerConfig "github.com/sds-framework/handler-lib/config"
)

type Client struct {
	socket *client.Socket
}

type ClientInterface interface {
	Close() error
	Timeout(duration time.Duration)
	Attempt(attempt uint8)

	StopService(depClient *clientConfig.Client) error
	AddService(service config.Service) error
	RemoveService(serviceName string) error
	StartService(serviceName string, parent *clientConfig.Client) (string, error)
	IsServiceRunning(depClient *clientConfig.Client) (bool, error)
}

func NewClient(runtimeSocket config.Socket) (*Client, error) {
	socketType := handlerConfig.SocketType(RuntimeSocketType)
	c := clientConfig.New("", runtimeSocket.Id, uint64(runtimeSocket.Port), socketType).
		UrlFunc(clientConfig.Url)

	socket, err := client.New(c)
	if err != nil {
		return nil, fmt.Errorf("client.New: %w", err)
	}

	return &Client{socket: socket}, nil
}

// Timeout of the client socket.
func (c *Client) Timeout(duration time.Duration) {
	c.socket.Timeout(duration)
}

// Attempt amount for requests.
func (c *Client) Attempt(attempt uint8) {
	c.socket.Attempt(attempt)
}

func (c *Client) Close() error {
	return c.socket.Close()
}

// StopService stops the running dependency service.
func (c *Client) StopService(depClient *clientConfig.Client) error {
	req := message.Request{
		Command: StopService,
		Parameters: key_value.New().
			Set("dep", depClient),
	}

	if c == nil {
		return fmt.Errorf("dep manager not initialized")
	}

	if c.socket == nil {
		return fmt.Errorf("dep manager socket was closed")
	}

	reply, err := c.socket.Request(&req)
	if err != nil {
		return fmt.Errorf("socket.Submit('%s'): %w", StopService, err)
	}

	if !reply.IsOK() {
		return fmt.Errorf("c.socket.Requeset(request='%v'): reply failed with: %s", req, reply.ErrorMessage())
	}

	return nil
}

// AddService registers a service in the runtime configuration.
func (c *Client) AddService(service config.Service) error {
	req := message.Request{
		Command: AddService,
		Parameters: key_value.New().
			Set("service", service),
	}

	reply, err := c.socket.Request(&req)
	if err != nil {
		return fmt.Errorf("socket.Submit('%s'): %w", AddService, err)
	}

	if !reply.IsOK() {
		return fmt.Errorf("reply.Message: %s", reply.ErrorMessage())
	}

	return nil
}

// RemoveService removes a service from the runtime configuration.
func (c *Client) RemoveService(serviceName string) error {
	req := message.Request{
		Command: RemoveService,
		Parameters: key_value.New().
			Set("service", serviceName),
	}

	reply, err := c.socket.Request(&req)
	if err != nil {
		return fmt.Errorf("socket.Submit('%s'): %w", RemoveService, err)
	}

	if !reply.IsOK() {
		return fmt.Errorf("reply.Message: %s", reply.ErrorMessage())
	}

	return nil
}

// StartService starts the dependency service and returns the generated runtime id.
func (c *Client) StartService(serviceName string, parent *clientConfig.Client) (string, error) {
	req := message.Request{
		Command: StartService,
		Parameters: key_value.New().
			Set("parent", parent).
			Set("service", serviceName),
	}

	reply, err := c.socket.Request(&req)
	if err != nil {
		return "", fmt.Errorf("socket.Submit('%s'): %w", StartService, err)
	}

	if !reply.IsOK() {
		return "", fmt.Errorf("reply.Message: %s", reply.ErrorMessage())
	}

	id, err := reply.ReplyParameters().StringValue("id")
	if err != nil {
		return "", fmt.Errorf("reply.Parameters.GetString('id'): %w", err)
	}

	return id, nil
}

// IsServiceRunning checks is the service running or not.
func (c *Client) IsServiceRunning(depClient *clientConfig.Client) (bool, error) {
	req := message.Request{
		Command: IsServiceRunning,
		Parameters: key_value.New().
			Set("dep", depClient),
	}

	reply, err := c.socket.Request(&req)
	if err != nil {
		return false, fmt.Errorf("socket.Request('%s'): %w", IsServiceRunning, err)
	}

	if !reply.IsOK() {
		return false, fmt.Errorf("reply.Message: %s", reply.ErrorMessage())
	}

	res, err := reply.ReplyParameters().BoolValue("running")
	if err != nil {
		return false, fmt.Errorf("reply.Parameters.GetBoolean('installed'): %w", err)
	}

	return res, nil
}
