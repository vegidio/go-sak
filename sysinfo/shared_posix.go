//go:build !windows

package sysinfo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func run(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolveTool(name), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		// %w rather than %s: callers need errors.Is(err, exec.ErrNotFound) to tell "this machine has no
		// nvidia-smi" apart from "nvidia-smi failed", and the stderr text alone cannot express that.
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%s: %s: %w", name, msg, err)
		}

		return nil, fmt.Errorf("%s: %w", name, err)
	}

	return out, nil
}
