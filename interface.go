package context

import "github.com/sds-framework/dev-lib/runtime"

type Interface interface {
	StartRuntimeHandler() error
	Runtime() runtime.ClientInterface
}
