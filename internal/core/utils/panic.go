package utils

import (
	"log"
	"runtime"
	"strings"

	"golang.org/x/net/context"
)

// temportary solution, pls dont judge
func trimStackPaths(stack []byte, folderName string) []byte {
	lines := strings.Split(string(stack), "\n")
	for i, line := range lines {
		idx := strings.Index(line, folderName)
		if idx != -1 {
			indentEnd := strings.LastIndex(line[:idx], "\t")
			if indentEnd == -1 {
				indentEnd = 0
			} else {
				indentEnd++
			}
			start := idx + len(folderName) + 1
			if start > len(line) {
				start = len(line)
			}
			lines[i] = line[:indentEnd] + line[start:]
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func CatchPanic() {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		stack = trimStackPaths(stack, "GoSally-mvp")
		log.Printf("recovered panic:\n%s", stack)
	}
}

func CatchPanicWithContext(ctx context.Context) {
	_, cancel := context.WithCancel(ctx)
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		stack = trimStackPaths(stack, "GoSally-mvp")
		log.Printf("recovered panic:\n%s", stack)
		cancel()
	}
}
