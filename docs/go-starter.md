# 🔷 Getting Started with Go Git Isolation

This guide provides the steps to initialize the Go project for Git isolation and implement the core functionality.

## 1. 📁 Project Setup

```bash
# Create project directory
mkdir -p git-isolation
cd git-isolation

# Initialize Go module
go mod init github.com/yourusername/git-isolation

# Create directory structure
mkdir -p cmd/gitiso
mkdir -p internal/{agent,docker,git,volume,config}
mkdir -p pkg/{api,models}
```

## 2. 📦 Install Dependencies

```bash
# Docker SDK
go get github.com/docker/docker/client
go get github.com/docker/docker/api/types
go get github.com/docker/docker/api/types/container
go get github.com/docker/docker/api/types/mount

# CLI framework
go get github.com/spf13/cobra
go get github.com/spf13/viper

# Logging
go get go.uber.org/zap
```

## 3. 🧩 Core Files Implementation

### pkg/models/agent.go

```go
package models

import (
	"time"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	// AgentStatusCreating indicates the agent is being created
	AgentStatusCreating AgentStatus = "creating"
	
	// AgentStatusReady indicates the agent is ready for use
	AgentStatusReady AgentStatus = "ready"
	
	// AgentStatusError indicates the agent is in an error state
	AgentStatusError AgentStatus = "error"
)

// Agent represents an isolated Git environment
type Agent struct {
	// ID is the unique identifier for the agent
	ID string `json:"id"`
	
	// Name is the display name for the agent
	Name string `json:"name"`
	
	// ContainerID is the Docker container ID
	ContainerID string `json:"containerId"`
	
	// Branch is the Git branch the agent is working on
	Branch string `json:"branch"`
	
	// HostDirectory is the directory on the host that is mounted to the container
	HostDirectory string `json:"hostDirectory"`
	
	// Status is the current status of the agent
	Status AgentStatus `json:"status"`
	
	// CreatedAt is the time the agent was created
	CreatedAt time.Time `json:"createdAt"`
}

// AgentConfig contains the configuration for creating a new agent
type AgentConfig struct {
	// ID is the unique identifier for the agent
	ID string `json:"id"`
	
	// Name is the display name for the agent
	Name string `json:"name"`
	
	// RepoURL is the URL of the Git repository
	RepoURL string `json:"repoUrl"`
	
	// BaseBranch is the branch to base the agent's branch on
	BaseBranch string `json:"baseBranch"`
	
	// BranchPrefix is the prefix to use for the agent's branch
	BranchPrefix string `json:"branchPrefix"`
	
	// HostDir is the directory on the host to mount to the container
	HostDir string `json:"hostDir"`
	
	// AuthorName is the Git author name
	AuthorName string `json:"authorName"`
	
	// AuthorEmail is the Git author email
	AuthorEmail string `json:"authorEmail"`
	
	// ShareAuth indicates whether to share the host's SSH keys
	ShareAuth bool `json:"shareAuth"`
}
```

### internal/docker/manager.go

```go
package docker

import (
	"context"
	"fmt"
	"io"
	
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ContainerConfig contains configuration for creating a container
type ContainerConfig struct {
	// Name is the name of the container
	Name string
	
	// Image is the Docker image to use
	Image string
	
	// HostDir is the directory on the host to mount
	HostDir string
	
	// ShareAuth indicates whether to share the host's SSH keys
	ShareAuth bool
	
	// SSHKeyPath is the path to the host's SSH keys
	SSHKeyPath string
	
	// AuthorName is the Git author name
	AuthorName string
	
	// AuthorEmail is the Git author email
	AuthorEmail string
	
	// CommitterName is the Git committer name
	CommitterName string
	
	// CommitterEmail is the Git committer email
	CommitterEmail string
}

// Manager handles Docker operations
type Manager struct {
	// client is the Docker client
	client *client.Client
}

// NewManager creates a new Docker manager
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	return &Manager{client: cli}, nil
}

// CreateContainer creates a new container for an agent
func (m *Manager) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	// Create volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: config.HostDir,
			Target: "/workspace",
		},
	}
	
	// Add SSH key mount if using shared auth
	if config.ShareAuth {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   config.SSHKeyPath,
			Target:   "/root/.ssh",
			ReadOnly: true,
		})
	}
	
	// Create container
	resp, err := m.client.ContainerCreate(
		ctx,
		&container.Config{
			Image: config.Image,
			Cmd:   []string{"sleep", "infinity"},
			Env: []string{
				fmt.Sprintf("GIT_AUTHOR_NAME=%s", config.AuthorName),
				fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", config.AuthorEmail),
				fmt.Sprintf("GIT_COMMITTER_NAME=%s", config.CommitterName),
				fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", config.CommitterEmail),
				"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no",
			},
		},
		&container.HostConfig{
			Mounts: mounts,
		},
		nil,
		nil,
		config.Name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	
	// Start container
	if err := m.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	
	return resp.ID, nil
}

// ExecCommand executes a command in a container
func (m *Manager) ExecCommand(ctx context.Context, containerID string, cmd []string) (string, string, error) {
	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	
	execID, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create exec: %w", err)
	}
	
	resp, err := m.client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()
	
	// Read output
	var outBuf, errBuf io.Writer
	outBuf = io.Writer(nil)
	errBuf = io.Writer(nil)
	
	var stdout, stderr string
	outputDone := make(chan error)
	
	go func() {
		_, err := stdcopy.StdCopy(outBuf, errBuf, resp.Reader)
		outputDone <- err
	}()
	
	select {
	case err := <-outputDone:
		if err != nil {
			return stdout, stderr, fmt.Errorf("error reading command output: %w", err)
		}
	case <-ctx.Done():
		return stdout, stderr, ctx.Err()
	}
	
	// Check exit code
	info, err := m.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return stdout, stderr, fmt.Errorf("failed to inspect exec: %w", err)
	}
	
	if info.ExitCode != 0 {
		return stdout, stderr, fmt.Errorf("command failed with exit code %d", info.ExitCode)
	}
	
	return stdout, stderr, nil
}

// RemoveContainer removes a container
func (m *Manager) RemoveContainer(ctx context.Context, containerID string) error {
	return m.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}
```

### internal/git/manager.go

```go
package git

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/yourusername/git-isolation/internal/docker"
)

// CloneOptions contains options for cloning a repository
type CloneOptions struct {
	// Depth is the number of commits to clone (0 for full history)
	Depth int
	
	// Reference is a path to a reference repository
	Reference string
}

// Manager handles Git operations
type Manager struct {
	docker *docker.Manager
}

// NewManager creates a new Git manager
func NewManager(dockerManager *docker.Manager) *Manager {
	return &Manager{
		docker: dockerManager,
	}
}

// Clone clones a repository in a container
func (m *Manager) Clone(ctx context.Context, containerID, repoURL, branch string, options CloneOptions) error {
	cmd := []string{"git", "clone"}
	
	if options.Depth > 0 {
		cmd = append(cmd, fmt.Sprintf("--depth=%d", options.Depth))
	}
	
	if options.Reference != "" {
		cmd = append(cmd, fmt.Sprintf("--reference=%s", options.Reference))
	}
	
	cmd = append(cmd, repoURL, "/workspace/repo")
	
	_, _, err := m.docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	
	return nil
}

// Checkout checks out a branch
func (m *Manager) Checkout(ctx context.Context, containerID, branch string, create bool) error {
	cmd := []string{"bash", "-c", "cd /workspace/repo && git checkout"}
	
	if create {
		cmd[2] = fmt.Sprintf("%s -b %s", cmd[2], branch)
	} else {
		cmd[2] = fmt.Sprintf("%s %s", cmd[2], branch)
	}
	
	_, _, err := m.docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}
	
	return nil
}

// ExecuteCommand executes a Git command in a container
func (m *Manager) ExecuteCommand(ctx context.Context, containerID, command string) (string, error) {
	cmd := []string{"bash", "-c", fmt.Sprintf("cd /workspace/repo && git %s", command)}
	
	stdout, stderr, err := m.docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return "", fmt.Errorf("git command failed: %s: %w", stderr, err)
	}
	
	return strings.TrimSpace(stdout), nil
}
```

### internal/agent/manager.go

```go
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	
	"github.com/yourusername/git-isolation/internal/docker"
	"github.com/yourusername/git-isolation/internal/git"
	"github.com/yourusername/git-isolation/pkg/models"
)

// Manager manages Git agents
type Manager struct {
	docker       *docker.Manager
	git          *git.Manager
	baseDir      string
	sshKeyPath   string
	agents       map[string]*models.Agent
	agentsMu     sync.RWMutex
	baseImage    string
	sharedObjects bool
}

// NewManager creates a new agent manager
func NewManager(dockerManager *docker.Manager, gitManager *git.Manager, baseDir, sshKeyPath, baseImage string) *Manager {
	return &Manager{
		docker:     dockerManager,
		git:        gitManager,
		baseDir:    baseDir,
		sshKeyPath: sshKeyPath,
		agents:     make(map[string]*models.Agent),
		baseImage:  baseImage,
	}
}

// EnableSharedObjects enables shared Git objects
func (m *Manager) EnableSharedObjects(enable bool) {
	m.sharedObjects = enable
}

// Create creates a new agent
func (m *Manager) Create(ctx context.Context, config *models.AgentConfig) (*models.Agent, error) {
	m.agentsMu.Lock()
	defer m.agentsMu.Unlock()
	
	// Check if agent already exists
	if _, exists := m.agents[config.ID]; exists {
		return nil, fmt.Errorf("agent %s already exists", config.ID)
	}
	
	// Create agent
	agent := &models.Agent{
		ID:        config.ID,
		Name:      config.Name,
		Status:    models.AgentStatusCreating,
		CreatedAt: time.Now(),
	}
	
	// Create host directory
	hostDir := filepath.Join(m.baseDir, fmt.Sprintf("agent-%s", config.ID))
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create host directory: %w", err)
	}
	agent.HostDirectory = hostDir
	
	// Create container
	containerName := fmt.Sprintf("git-agent-%s", config.ID)
	containerID, err := m.docker.CreateContainer(ctx, &docker.ContainerConfig{
		Name:          containerName,
		Image:         m.baseImage,
		HostDir:       hostDir,
		ShareAuth:     config.ShareAuth,
		SSHKeyPath:    m.sshKeyPath,
		AuthorName:    config.AuthorName,
		AuthorEmail:   config.AuthorEmail,
		CommitterName: config.AuthorName,
		CommitterEmail: config.AuthorEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	agent.ContainerID = containerID
	
	// Clone repository
	cloneOpts := git.CloneOptions{}
	if m.sharedObjects {
		cloneOpts.Reference = "/shared-objects"
	}
	
	if err := m.git.Clone(ctx, containerID, config.RepoURL, config.BaseBranch, cloneOpts); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	
	// Create branch
	branchName := fmt.Sprintf("%s%s", config.BranchPrefix, config.ID)
	if err := m.git.Checkout(ctx, containerID, branchName, true); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}
	agent.Branch = branchName
	
	// Update status
	agent.Status = models.AgentStatusReady
	m.agents[config.ID] = agent
	
	// Create status file
	m.updateStatusFile(ctx, agent)
	
	return agent, nil
}

// Get gets an agent by ID
func (m *Manager) Get(id string) (*models.Agent, error) {
	m.agentsMu.RLock()
	defer m.agentsMu.RUnlock()
	
	agent, exists := m.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", id)
	}
	
	return agent, nil
}

// ExecuteGitCommand executes a Git command for an agent
func (m *Manager) ExecuteGitCommand(ctx context.Context, agentID, command string) (string, error) {
	agent, err := m.Get(agentID)
	if err != nil {
		return "", err
	}
	
	output, err := m.git.ExecuteCommand(ctx, agent.ContainerID, command)
	if err != nil {
		return "", err
	}
	
	// Update status file
	m.updateStatusFile(ctx, agent)
	
	return output, nil
}

// updateStatusFile updates the status file for an agent
func (m *Manager) updateStatusFile(ctx context.Context, agent *models.Agent) {
	statusPath := filepath.Join(agent.HostDirectory, ".git-status.md")
	
	// Get branch and status
	branch, _ := m.git.ExecuteCommand(ctx, agent.ContainerID, "branch -v")
	status, _ := m.git.ExecuteCommand(ctx, agent.ContainerID, "status")
	
	// Create status file
	content := fmt.Sprintf("# Git Status for %s (%s)\n", agent.Name, agent.Branch)
	content += fmt.Sprintf("Last updated: %s\n\n", time.Now().Format(time.RFC3339))
	content += "## Current Branch\n"
	content += branch + "\n\n"
	content += "## Status\n"
	content += status + "\n"
	
	os.WriteFile(statusPath, []byte(content), 0644)
}

// Remove removes an agent
func (m *Manager) Remove(ctx context.Context, agentID string) error {
	m.agentsMu.Lock()
	defer m.agentsMu.Unlock()
	
	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}
	
	// Remove container
	if err := m.docker.RemoveContainer(ctx, agent.ContainerID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	
	// Remove from agents map
	delete(m.agents, agentID)
	
	return nil
}
```

### cmd/gitiso/main.go

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/spf13/cobra"
	"github.com/yourusername/git-isolation/internal/agent"
	"github.com/yourusername/git-isolation/internal/docker"
	"github.com/yourusername/git-isolation/internal/git"
	"github.com/yourusername/git-isolation/pkg/models"
)

func main() {
	var (
		repoURL        string
		baseBranch     string
		hostProjectDir string
		sshKeyPath     string
		shareAuth      bool
		baseImage      string
		enableSharedObjects bool
	)
	
	// Root command
	rootCmd := &cobra.Command{
		Use:   "gitiso",
		Short: "Git Isolation - Isolated Git environments",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Ensure project directory exists
			if err := os.MkdirAll(hostProjectDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating project directory: %v\n", err)
				os.Exit(1)
			}
		},
	}
	
	// Create command
	createCmd := &cobra.Command{
		Use:   "create [agent-id]",
		Short: "Create a new isolated Git environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			
			// Create Docker manager
			dockerManager, err := docker.NewManager()
			if err != nil {
				return fmt.Errorf("failed to create Docker manager: %w", err)
			}
			
			// Create Git manager
			gitManager := git.NewManager(dockerManager)
			
			// Create agent manager
			agentManager := agent.NewManager(dockerManager, gitManager, hostProjectDir, sshKeyPath, baseImage)
			agentManager.EnableSharedObjects(enableSharedObjects)
			
			// Create agent
			agent, err := agentManager.Create(context.Background(), &models.AgentConfig{
				ID:           agentID,
				Name:         fmt.Sprintf("Agent %s", agentID),
				RepoURL:      repoURL,
				BaseBranch:   baseBranch,
				BranchPrefix: "feature/agent-",
				AuthorName:   "Git Isolation",
				AuthorEmail:  "git-isolation@example.com",
				ShareAuth:    shareAuth,
			})
			if err != nil {
				return fmt.Errorf("failed to create agent: %w", err)
			}
			
			fmt.Printf("✅ Agent %s created successfully\n", agentID)
			fmt.Printf("   - Container: %s\n", agent.ContainerID)
			fmt.Printf("   - Branch: %s\n", agent.Branch)
			fmt.Printf("   - Files: %s\n", agent.HostDirectory)
			
			return nil
		},
	}
	
	// Exec command
	execCmd := &cobra.Command{
		Use:   "exec [agent-id] [git-command]",
		Short: "Execute a Git command in an isolated environment",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			gitCommand := args[1]
			for i := 2; i < len(args); i++ {
				gitCommand += " " + args[i]
			}
			
			// Create Docker manager
			dockerManager, err := docker.NewManager()
			if err != nil {
				return fmt.Errorf("failed to create Docker manager: %w", err)
			}
			
			// Create Git manager
			gitManager := git.NewManager(dockerManager)
			
			// Create agent manager
			agentManager := agent.NewManager(dockerManager, gitManager, hostProjectDir, sshKeyPath, baseImage)
			
			// Execute Git command
			output, err := agentManager.ExecuteGitCommand(context.Background(), agentID, gitCommand)
			if err != nil {
				return fmt.Errorf("failed to execute Git command: %w", err)
			}
			
			fmt.Println(output)
			
			return nil
		},
	}
	
	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation
			return nil
		},
	}
	
	// Remove command
	removeCmd := &cobra.Command{
		Use:   "remove [agent-id]",
		Short: "Remove an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation
			return nil
		},
	}
	
	// Add flags
	rootCmd.PersistentFlags().StringVar(&repoURL, "repo-url", "", "Repository URL")
	rootCmd.PersistentFlags().StringVar(&baseBranch, "branch", "main", "Base branch")
	rootCmd.PersistentFlags().StringVar(&hostProjectDir, "project-dir", filepath.Join(os.Getenv("HOME"), "git-isolation", "project-files"), "Host project directory")
	rootCmd.PersistentFlags().StringVar(&sshKeyPath, "ssh-path", filepath.Join(os.Getenv("HOME"), ".ssh"), "SSH key path")
	rootCmd.PersistentFlags().BoolVar(&shareAuth, "share-auth", true, "Share host authentication")
	rootCmd.PersistentFlags().StringVar(&baseImage, "image", "ubuntu:latest", "Base Docker image")
	rootCmd.PersistentFlags().BoolVar(&enableSharedObjects, "shared-objects", false, "Enable shared Git objects")
	
	// Add commands
	rootCmd.AddCommand(createCmd, execCmd, listCmd, removeCmd)
	
	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

## 4. 🚀 Building and Running

```bash
# Build the CLI
cd git-isolation
go build -o bin/gitiso ./cmd/gitiso

# Create a test agent
./bin/gitiso --repo-url https://github.com/yourusername/yourrepo.git create test-agent

# Execute a Git command
./bin/gitiso exec test-agent status
```

## 5. 📚 Next Steps

1. **Add advanced features**:
   - Implement shared object pool
   - Add shallow cloning
   - Implement sparse checkout

2. **Improve error handling**:
   - Add better error reporting
   - Implement logging with zap

3. **Add HTTP API server**:
   - Implement the HTTP API in `cmd/gitisoserver`
   - Add REST endpoints for all operations

4. **Add resource monitoring**:
   - Implement the stats API
   - Add periodic monitoring

## 6. 📦 Example Docker Image Creation

Create a `Dockerfile` to build a suitable base image:

```dockerfile
FROM ubuntu:latest

# Install required packages
RUN apt-get update && apt-get install -y \
    git \
    openssh-client \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set up SSH
RUN mkdir -p /root/.ssh && \
    chmod 700 /root/.ssh && \
    echo "StrictHostKeyChecking no" >> /etc/ssh/ssh_config

# Default working directory
WORKDIR /workspace

# Default command
CMD ["sleep", "infinity"]
```

Build with:
```bash
docker build -t git-isolation-base .
```

Then use with:
```bash
./bin/gitiso --image git-isolation-base create test-agent
```

This setup provides a solid foundation for building a production-ready Git isolation system in Go! 