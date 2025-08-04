package utils

import (
	"log"
	"runtime"

	"golang.org/x/net/context"
)

func CatchPanic() {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		log.Printf("recovered panic:\n%s", stack)
	}
}

func CatchPanicWithCancel(cancel context.CancelFunc) {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		log.Printf("recovered panic:\n%s", stack)
		cancel()
	}
}

func CatchPanicWithFallback(onPanic func(any)) {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		log.Printf("recovered panic:\n%s", stack)
		onPanic(err)
	}
}
