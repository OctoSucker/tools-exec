package exec

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type DockerExecutor struct {
	Binary  string
	Image   string
	Network string
}

func (e *DockerExecutor) Run(ctx context.Context, argv []string, workDir string, env []string, timeoutSec int, limits SandboxLimits) (*RunResult, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	args := []string{"run", "--rm", "-v", workDir + ":/work", "-w", "/work"}
	if limits.Network == "none" {
		args = append(args, "--network=none")
	}
	for _, s := range env {
		args = append(args, "-e", s)
	}
	args = append(args, e.Image)
	args = append(args, argv...)

	cmd := exec.CommandContext(runCtx, e.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.Bytes()
	errOut := stderr.Bytes()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}

	result := &RunResult{
		Stdout:   out,
		Stderr:   errOut,
		ExitCode: exitCode,
	}
	if runCtx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		result.Message = "command timed out and container was killed"
	}
	if err != nil && !result.Timeout {
		result.Message = err.Error()
	}
	return result, nil
}

func NewDockerExecutor(binary, image, network string) *DockerExecutor {
	if binary == "" {
		binary = "docker"
	}
	return &DockerExecutor{Binary: binary, Image: image, Network: network}
}
