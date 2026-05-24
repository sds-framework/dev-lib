// Package context sets up the developer context.
package context

import (
	"fmt"

	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/dev-lib/dep_client"
	"github.com/sds-framework/dev-lib/dep_handler"
	"github.com/sds-framework/handler-lib/manager_client"
)

// A Context handles the config of the contexts
type Context struct {
	Config                config.SdsService
	runtimeHandler        *dep_handler.DepHandler
	runtimeHandlerManager manager_client.Interface
	runtimeClient         *dep_client.Client
}

// New creates a developer context and loads it with the dev configuration.
func New(configPath string) (*Context, error) {
	ctx := &Context{}

	appConfig, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.Load('%s'): %w", configPath, err)
	}
	ctx.Config = appConfig

	return ctx, nil
}

func (ctx *Context) IsHandlerRunning() bool {
	return ctx.runtimeHandlerManager != nil && ctx.runtimeHandler != nil
}

func (ctx *Context) Runtime() dep_client.Interface {
	return ctx.runtimeClient
}

// CloseRuntimeHandler shuts down the runtime handler through its manager client.
func (ctx *Context) CloseRuntimeHandler() error {
	if ctx.runtimeHandlerManager != nil {
		if err := ctx.runtimeHandlerManager.Close(); err != nil {
			return fmt.Errorf("ctx.runtimeHandlerManager.Close: %w", err)
		}
		ctx.runtimeHandlerManager = nil
	}

	return nil
}

// StartRuntimeHandler starts the runtime handler.
func (ctx *Context) StartRuntimeHandler() error {
	if ctx.runtimeHandlerManager != nil {
		return fmt.Errorf("runtime handler already started")
	}

	var err error
	ctx.runtimeHandler, err = dep_handler.New(&ctx.Config)
	if err != nil {
		return fmt.Errorf("dep_handler.New: %w", err)
	}

	err = ctx.runtimeHandler.Start()
	if err != nil {
		return fmt.Errorf("runtimeHandler: %w", err)
	}

	ctx.runtimeHandlerManager, err = manager_client.New(dep_handler.ServiceConfig())
	if err != nil {
		return fmt.Errorf("manager_client.New('dep_handler'): %w", err)
	}

	runtimeAccess, err := dep_client.New()
	if err != nil {
		return fmt.Errorf("dep_client.New: %w", err)
	}

	ctx.runtimeClient = runtimeAccess

	return nil
}
