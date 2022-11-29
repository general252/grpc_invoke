package browsers

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Open(uri string) error {
	// 不同平台启动指令不同
	var commands = map[string]string{
		"windows": "explorer",
		"darwin":  "open",
		"linux":   "xdg-open",
	}

	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	cmd := exec.Command(run, uri)
	return cmd.Run()
}
