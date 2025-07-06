package init

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func init() {
	if strings.HasPrefix(os.Args[0], "/tmp") {
		return
	}
	runPath, err := os.MkdirTemp("", "*-gs-runtime")
	log.SetOutput(os.Stderr)
	input, err := os.Open(os.Args[0])
	if err != nil {
		log.Fatalf("Failed to init node: %s", err)
	}

	runBinaryPath := filepath.Join(runPath, "node")
	output, err := os.Create(runBinaryPath)
	if err != nil {
		log.Fatalf("Failed to init node: %s", err)
	}

	if _, err := io.Copy(output, input); err != nil {
		log.Fatalf("Failed to init node: %s", err)
	}

	// Делаем исполняемым (на всякий случай)
	if err := os.Chmod(runBinaryPath, 0755); err != nil {
		log.Fatalf("Failed to init node: %s", err)
	}

	input.Close()
	output.Close()
	runArgs := os.Args
	runArgs[0] = runBinaryPath
	if err := syscall.Exec(runBinaryPath, runArgs, append(os.Environ(), "GS_RUNTIME_PATH="+runPath)); err != nil {
		log.Fatalf("Failed to init node: %s", err)
	}
}
