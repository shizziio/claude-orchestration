// Package compose manages Docker Compose environments for DevFlow projects.
package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// ServiceStatus represents the state of a single Docker Compose service.
type ServiceStatus struct {
	Name   string `json:"name"`
	State  string `json:"state"`  // running | exited | paused | ...
	Status string `json:"status"` // "Up 2 minutes", "Exited (0) 5 minutes ago"
	Ports  string `json:"ports"`
}

// Up starts services via `docker compose up -d`.
func Up(ctx context.Context, projectDir string, composeFile string, overrideFile string, envVars map[string]string) error {
	args := composeArgs(composeFile, overrideFile)
	args = append(args, "up", "-d", "--remove-orphans")
	return run(ctx, projectDir, envVars, args...)
}

// Down stops and removes services via `docker compose down`.
func Down(ctx context.Context, projectDir string, composeFile string, overrideFile string) error {
	args := composeArgs(composeFile, overrideFile)
	args = append(args, "down", "--remove-orphans")
	return run(ctx, projectDir, nil, args...)
}

// Status returns the status of each service via `docker compose ps --format json`.
func Status(ctx context.Context, projectDir string, composeFile string, overrideFile string) ([]ServiceStatus, error) {
	args := composeArgs(composeFile, overrideFile)
	args = append(args, "ps", "--format", "json")

	out, err := output(ctx, projectDir, args...)
	if err != nil {
		return nil, fmt.Errorf("docker compose ps: %w\noutput: %s", err, out)
	}

	if strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}

	// docker compose ps --format json outputs one JSON object per line
	var services []ServiceStatus
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var raw struct {
			Name    string `json:"Name"`
			Service string `json:"Service"`
			State   string `json:"State"`
			Status  string `json:"Status"`
			Ports   string `json:"Ports"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		name := raw.Service
		if name == "" {
			name = raw.Name
		}
		services = append(services, ServiceStatus{
			Name:   name,
			State:  raw.State,
			Status: raw.Status,
			Ports:  raw.Ports,
		})
	}
	return services, nil
}

// Logs streams logs for a service. If service is empty, streams all services.
// The writer receives log output until ctx is cancelled.
func Logs(ctx context.Context, projectDir string, composeFile string, overrideFile string, service string, tail int, w io.Writer) error {
	args := composeArgs(composeFile, overrideFile)
	args = append(args, "logs", "--no-log-prefix", "-f")
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	if service != "" {
		args = append(args, service)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = projectDir
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func composeArgs(composeFile, overrideFile string) []string {
	args := []string{"compose"}
	if composeFile != "" {
		args = append(args, "-f", composeFile)
	}
	if overrideFile != "" {
		args = append(args, "-f", overrideFile)
	}
	return args
}

func run(ctx context.Context, dir string, envVars map[string]string, args ...string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	if len(envVars) > 0 {
		cmd.Env = append(cmd.Environ(), mapToEnv(envVars)...)
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\noutput: %s", err, buf.String())
	}
	return nil
}

func output(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.Bytes(), err
	}
	return stdout.Bytes(), nil
}

func mapToEnv(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
