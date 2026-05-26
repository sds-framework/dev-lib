package context

import "github.com/noPerfection/context/runtime"

type Interface interface {
	StartRuntimeHandler() error
	Runtime() runtime.ClientInterface
}
