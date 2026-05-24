package context

import "github.com/sds-framework/dev-lib/dep_client"

type Interface interface {
	Type() ContextType
	StartDepManager() error
	Close() error // Close the dep handler. The dep manager client is not closed.
	IsRunning() bool
	IsDepManagerRunning() bool
	SetDepClient(p dep_client.Interface) error
	DepClient() dep_client.Interface
}
