package context

import "github.com/sds-framework/context/runtime"

type Interface interface {
	StartRuntimeHandler() error
	Runtime() runtime.ClientInterface
}
