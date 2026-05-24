// Package context sets up the developer context.
package context

import (
	"fmt"

	config "github.com/sds-framework/config-lib"
	"github.com/sds-framework/dev-lib/runtime"
)

// A Context handles the config of the contexts
type Context struct {
	Config         config.SdsService
	runtimeHandler *runtime.Handler
	runtimeClient  *runtime.Client
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

func (ctx *Context) Runtime() runtime.ClientInterface {
	return ctx.runtimeClient
}

// StartRuntimeHandler starts the runtime handler.
func (ctx *Context) StartRuntimeHandler() error {
	if ctx.runtimeHandler != nil {
		return fmt.Errorf("runtime handler already started")
	}

	var err error
	ctx.runtimeHandler, err = runtime.NewHandler(&ctx.Config)
	if err != nil {
		return fmt.Errorf("runtime.NewHandler: %w", err)
	}

	err = ctx.runtimeHandler.Start()
	if err != nil {
		return fmt.Errorf("runtimeHandler: %w", err)
	}

	runtimeAccess, err := runtime.NewClient()
	if err != nil {
		return fmt.Errorf("runtime.NewClient: %w", err)
	}

	ctx.runtimeClient = runtimeAccess

	return nil
}
