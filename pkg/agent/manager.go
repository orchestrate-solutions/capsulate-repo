package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

// AgentConfig holds configuration for a git-isolate agent
type AgentConfig struct {
	ID              string
	DependencyLevel string
	OverrideDeps    []string
	UseOverlay      bool
}

// Manager handles agent lifecycle operations
type Manager struct {
	dockerClient  *client.Client
	baseImageName string
	sshDir        string
	workspaceDir  string
}

// NewManager creates a new agent manager
func NewManager() *Manager {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Errorf("failed to create Docker client: %v", err))
	}

	// Get SSH directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get user home directory: %v", err))
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	// Get current working directory as workspace
	workspaceDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get current working directory: %v", err))
	}

	return &Manager{
		dockerClient:  cli,
		baseImageName: "capsulate-base:latest",
		sshDir:        sshDir,
		workspaceDir:  workspaceDir,
	}
}

// Create creates a new agent container
func (m *Manager) Create(config AgentConfig) error {
	ctx := context.Background()

	// Ensure base image exists
	m.ensureBaseImage(ctx)

	// Container name based on agent ID
	containerName := fmt.Sprintf("capsulate-%s", config.ID)

	// Check if container already exists
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				return fmt.Errorf("agent with ID '%s' already exists", config.ID)
			}
		}
	}

	// Prepare volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: m.sshDir,
			Target: "/root/.ssh",
			ReadOnly: true, // Mount SSH directory as read-only
		},
		{
			Type:   mount.TypeBind,
			Source: m.workspaceDir,
			Target: "/workspace",
		},
	}

	// Create container
	resp, err := m.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: m.baseImageName,
			Cmd:   []string{"tail", "-f", "/dev/null"}, // Keep container running
			Tty:   true,
			Env: []string{
				fmt.Sprintf("AGENT_ID=%s", config.ID),
				fmt.Sprintf("DEPENDENCY_LEVEL=%s", config.DependencyLevel),
				fmt.Sprintf("OVERRIDE_DEPS=%s", strings.Join(config.OverrideDeps, ",")),
				fmt.Sprintf("USE_OVERLAY=%v", config.UseOverlay),
			},
		},
		&container.HostConfig{
			Mounts: mounts,
		},
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Start container
	if err := m.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	// Setup initial environment if needed (future enhancement)
	// This is where we'd initialize Git repos, etc.

	return nil
}

// Exec executes a command in the agent container
func (m *Manager) Exec(agentID string, command string) (string, error) {
	ctx := context.Background()
	containerName := fmt.Sprintf("capsulate-%s", agentID)

	// Find container by name
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %v", err)
	}

	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				containerID = c.ID
				break
			}
		}
	}

	if containerID == "" {
		return "", fmt.Errorf("agent '%s' not found or not running", agentID)
	}

	// Create exec configuration
	execConfig := types.ExecConfig{
		Cmd:          []string{"/bin/sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create exec instance
	execID, err := m.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %v", err)
	}

	// Start exec instance
	resp, err := m.dockerClient.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to start exec: %v", err)
	}
	defer resp.Close()

	// Read the output
	var stdout bytes.Buffer
	if _, err := io.Copy(&stdout, resp.Reader); err != nil {
		return "", fmt.Errorf("failed to read exec output: %v", err)
	}

	// Get exec exit code
	inspectResp, err := m.dockerClient.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec: %v", err)
	}

	// Check exit code
	if inspectResp.ExitCode != 0 {
		return stdout.String(), fmt.Errorf("command exited with code %d", inspectResp.ExitCode)
	}

	return stdout.String(), nil
}

// Destroy stops and removes an agent container
func (m *Manager) Destroy(agentID string) error {
	ctx := context.Background()
	containerName := fmt.Sprintf("capsulate-%s", agentID)

	// Find container by name
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				containerID = c.ID
				break
			}
		}
	}

	if containerID == "" {
		return fmt.Errorf("agent '%s' not found", agentID)
	}

	// Stop container if it's running
	timeout := 10 * time.Second
	if err := m.dockerClient.ContainerStop(ctx, containerID, &timeout); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}

	// Remove container
	if err := m.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container: %v", err)
	}

	return nil
}

// ensureBaseImage makes sure the base Docker image exists
func (m *Manager) ensureBaseImage(ctx context.Context) error {
	// Check if image exists
	images, err := m.dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %v", err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == m.baseImageName {
				return nil // Image exists
			}
		}
	}

	// If we're here, we need to build the image
	fmt.Println("Building base image...")
	
	// TODO: Implement actual image building
	// For now, use a simple Ubuntu image with Git installed
	// In a real implementation, we would build a Dockerfile
	
	// Pull ubuntu image
	out, err := m.dockerClient.ImagePull(ctx, "ubuntu:latest", types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull ubuntu image: %v", err)
	}
	defer out.Close()
	io.Copy(io.Discard, out) // Discard output
	
	// Create a container to install Git
	tempContainerName := "capsulate-image-builder"
	resp, err := m.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"/bin/bash", "-c", "apt-get update && apt-get install -y git"},
		},
		nil,
		nil,
		nil,
		tempContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create temp container: %v", err)
	}
	
	// Start container
	if err := m.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start temp container: %v", err)
	}
	
	// Wait for container to finish
	statusCh, errCh := m.dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("container wait error: %v", err)
		}
	case <-statusCh:
	}
	
	// Commit the container as our base image
	_, err = m.dockerClient.ContainerCommit(ctx, resp.ID, types.ContainerCommitOptions{
		Reference: m.baseImageName,
	})
	if err != nil {
		return fmt.Errorf("failed to commit container: %v", err)
	}
	
	// Remove the temporary container
	if err := m.dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove temp container: %v", err)
	}
	
	fmt.Println("Base image built successfully")
	return nil
} 