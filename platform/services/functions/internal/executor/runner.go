package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	imageTypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"

	"platform/functions/internal/domain/function"
)

type Runner interface {
	Run(ctx context.Context, fn *function.Function) (string, error)
}

type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	dcli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &DockerRunner{cli: dcli}, nil
}

func (dr *DockerRunner) Run(ctx context.Context, fn *function.Function) (string, error) {
	var image string
	var cmd []string

	switch strings.ToLower(fn.Language) {
	case "python":
		image = "python:3.10-alpine"
		cmd = []string{"python", "-c", fn.Code}
	case "go":
		image = "golang:1.21-alpine"
		cmd = []string{"sh", "-c", fmt.Sprintf("echo '%s' && echo 'Pretend to run Go code'", fn.Code)}
	default:
		image = "alpine"
		cmd = []string{"echo", fn.Code}
	}

	containerName := fmt.Sprintf("fn-%s-%s", fn.Language, uuid.New().String()[:8])

	// Container config
	containerCfg := &container.Config{
		Image: image,
		Cmd:   cmd,
		Tty:   false,
	}

	if err := dr.pullIfNotExists(ctx, image); err != nil {
		return "", err
	}

	resp, err := dr.cli.ContainerCreate(ctx, containerCfg, nil, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("container create error: %w", err)
	}
	containerID := resp.ID

	if err := dr.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("container start error: %w", err)
	}

	waitCh, errCh := dr.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case status := <-waitCh:
		if status.Error != nil {
			return "", fmt.Errorf("container wait error: %s", status.Error.Message)
		}
		if status.StatusCode != 0 {
			return "", fmt.Errorf("container exit code %d", status.StatusCode)
		}
	case e := <-errCh:
		if e != nil {
			return "", fmt.Errorf("container wait error: %w", e)
		}
	case <-ctx.Done():
		// Context cancelled, stop the container
		timeout := 0 // no grace period
		if err := dr.cli.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeout}); err != nil {
			return "", fmt.Errorf("failed to stop container: %w", err)
		}
		return "", ctx.Err()
	}

	logs, err := dr.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("container logs error: %w", err)
	}
	defer logs.Close()

	logOutput, err := readLogs(logs)
	if err != nil {
		return "", err
	}

	_ = dr.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	})

	return logOutput, nil
}

func (dr *DockerRunner) pullIfNotExists(ctx context.Context, image string) error {
	_, _, err := dr.cli.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}

	pullResp, err := dr.cli.ImagePull(ctx, image, imageTypes.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull error: %w", err)
	}
	defer pullResp.Close()

	_, _ = io.Copy(io.Discard, pullResp)
	return nil
}

func readLogs(reader io.ReadCloser) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	_, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader)
	if err != nil {
		return "", err
	}

	combined := stdoutBuf.String() + stderrBuf.String()

	for _, ctrl := range []string{"\x00", "\x01", "\x02", "\x03", "\x04",
		"\x05", "\x06", "\x07", "\x08", "\x0B", "\x0C",
		"\x0E", "\x0F"} {
		combined = strings.ReplaceAll(combined, ctrl, "")
	}

	combined = strings.ReplaceAll(combined, "\r\n", "\\n")
	combined = strings.ReplaceAll(combined, "\r", "\\n")
	combined = strings.ReplaceAll(combined, "\n", "\\n")

	return combined, nil
}
