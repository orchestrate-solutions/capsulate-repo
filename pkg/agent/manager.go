package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

// AgentConfig holds configuration for a git-isolate agent
type AgentConfig struct {
	ID              string
	DependencyLevel string // "core", "team", or "container"
	TeamID          string // Team identifier for team-level dependencies
	OverrideDeps    []string
	UseOverlay      bool
	// Git repository configuration
	RepoURL         string // URL of Git repository to clone
	Branch          string // Branch to checkout
	Depth           int    // Depth for shallow clones
	GitConfig       map[string]string // Git configuration to apply
}

// GitStatus represents the status of a Git repository in an agent
type GitStatus struct {
	Branch          string
	CurrentCommit   string
	ModifiedFiles   []string
	UntrackedFiles  []string
	AheadCount      int
	BehindCount     int
}

// Manager manages Docker containers for git-isolate agents
type Manager struct {
	dockerClient  *client.Client
	baseImageName string
	sshDir        string
	workspaceDir  string
	// Dependency and file system management
	coreDepsPath     string
	teamDepsPath     map[string]string
	containerDepsPath string
	// OverlayFS paths
	baseRepoPath     string
	diffsPath        string
	workPath         string
}

// NewManager creates a new Manager instance
func NewManager(sshDir, workspaceDir string) (*Manager, error) {
	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}

	// Initialize manager
	m := &Manager{
		dockerClient:     dockerClient,
		baseImageName:    "capsulate-base:latest",
		sshDir:           sshDir,
		workspaceDir:     workspaceDir,
		// Default paths for dependency management
		coreDepsPath:     filepath.Join(workspaceDir, ".capsulate", "dependencies", "core"),
		teamDepsPath:     make(map[string]string),
		containerDepsPath: filepath.Join(workspaceDir, ".capsulate", "dependencies", "container"),
		// Default paths for OverlayFS
		baseRepoPath:     filepath.Join(workspaceDir, ".capsulate", "overlay", "base"),
		diffsPath:        filepath.Join(workspaceDir, ".capsulate", "overlay", "diffs"),
		workPath:         filepath.Join(workspaceDir, ".capsulate", "overlay", "work"),
	}

	// Ensure directories exist
	dirs := []string{
		m.coreDepsPath,
		m.containerDepsPath,
		m.baseRepoPath,
		m.diffsPath,
		m.workPath,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return m, nil
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

	// Create agent-specific workspace directory
	agentWorkspace := filepath.Join(m.workspaceDir, ".capsulate", "workspaces", config.ID)
	if err := os.MkdirAll(agentWorkspace, 0755); err != nil {
		return fmt.Errorf("failed to create agent workspace directory: %v", err)
	}

	// Prepare volume mounts
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   m.sshDir,
			Target:   "/root/.ssh",
			ReadOnly: true, // Mount SSH directory as read-only
		},
	}
	
	// Add workspace mount - either direct or via overlay
	if config.UseOverlay {
		// For overlay, we'll set up the separate mounts for base, diff, and work dirs
		// Create container-specific directories
		containerDiffPath := filepath.Join(m.diffsPath, config.ID)
		containerWorkPath := filepath.Join(m.workPath, config.ID)
		
		os.MkdirAll(containerDiffPath, 0755)
		os.MkdirAll(containerWorkPath, 0755)
		
		// Mount the base repo as read-only
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.baseRepoPath,
			Target:   "/workspace/base",
			ReadOnly: true,
		})
		
		// Mount container-specific diff directory
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: containerDiffPath,
			Target: "/workspace/diff",
		})
		
		// Mount container-specific work directory
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: containerWorkPath,
			Target: "/workspace/work",
		})
		
		// We'll need an entrypoint script to set up the overlay mount inside container
	} else {
		// Without overlay, mount workspace directory directly
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: agentWorkspace,
			Target: "/workspace",
		})
	}
	
	// Add dependency mounts based on isolation level
	// Always mount core deps if available
	if _, err := os.Stat(m.coreDepsPath); err == nil {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.coreDepsPath,
			Target:   "/workspace/core-deps",
			ReadOnly: true,
		})
	}
	
	// Add team deps if applicable
	if config.DependencyLevel == "team" && config.TeamID != "" {
		teamPath, exists := m.teamDepsPath[config.TeamID]
		if !exists {
			// Create team dependency path if it doesn't exist
			teamPath = filepath.Join(m.workspaceDir, ".capsulate", "dependencies", "team", config.TeamID)
			os.MkdirAll(teamPath, 0755)
			m.teamDepsPath[config.TeamID] = teamPath
		}
		
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   teamPath,
			Target:   "/workspace/team-deps",
			ReadOnly: true,
		})
	}
	
	// Add container-specific deps directory
	containerDepsPath := filepath.Join(m.containerDepsPath, config.ID)
	os.MkdirAll(containerDepsPath, 0755)
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: containerDepsPath,
		Target: "/workspace/container-deps",
	})

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("AGENT_ID=%s", config.ID),
		fmt.Sprintf("DEPENDENCY_LEVEL=%s", config.DependencyLevel),
		fmt.Sprintf("TEAM_ID=%s", config.TeamID),
		fmt.Sprintf("OVERRIDE_DEPS=%s", strings.Join(config.OverrideDeps, ",")),
		fmt.Sprintf("USE_OVERLAY=%v", config.UseOverlay),
	}

	// Create container
	resp, err := m.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: m.baseImageName,
			Cmd:   []string{"tail", "-f", "/dev/null"}, // Keep container running
			Tty:   true,
			Env:   env,
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

	// Set up the overlay filesystem if requested
	if config.UseOverlay {
		setupCmd := `mkdir -p /workspace/merged && 
			mount -t overlay overlay -o lowerdir=/workspace/base,upperdir=/workspace/diff,workdir=/workspace/work /workspace/merged &&
			mkdir -p /workspace/merged/repo`
		_, err := m.Exec(config.ID, setupCmd)
		if err != nil {
			return fmt.Errorf("failed to set up overlay filesystem: %v", err)
		}
	} else {
		// Ensure repo directory exists
		_, err := m.Exec(config.ID, "mkdir -p /workspace/repo")
		if err != nil {
			return fmt.Errorf("failed to create repo directory: %v", err)
		}
	}

	// Set up dependency linking
	depSetupCmd := m.generateDependencySetupScript(config)
	_, err = m.Exec(config.ID, depSetupCmd)
	if err != nil {
		return fmt.Errorf("failed to set up dependencies: %v", err)
	}

	// Setup Git repository if URL is provided
	if config.RepoURL != "" {
		return m.setupGitRepository(config)
	}

	return nil
}

// generateDependencySetupScript creates a script to set up the dependencies inside the container
func (m *Manager) generateDependencySetupScript(config AgentConfig) string {
	script := `#!/bin/bash
# Set up node_modules directory
mkdir -p /workspace/node_modules

# Link core dependencies if available
if [ -d "/workspace/core-deps" ]; then
    for pkg in $(find /workspace/core-deps -maxdepth 1 -type d ! -path "/workspace/core-deps"); do
        pkg_name=$(basename $pkg)
        # Don't link if it's in the override list
        if [[ ! " %s " =~ " $pkg_name " ]]; then
            ln -sf "$pkg" "/workspace/node_modules/$pkg_name"
        fi
    done
fi

# Link team dependencies if available
if [ "%s" = "team" ] && [ -d "/workspace/team-deps" ]; then
    for pkg in $(find /workspace/team-deps -maxdepth 1 -type d ! -path "/workspace/team-deps"); do
        pkg_name=$(basename $pkg)
        # Don't link if it's in the override list
        if [[ ! " %s " =~ " $pkg_name " ]]; then
            ln -sf "$pkg" "/workspace/node_modules/$pkg_name"
        fi
    done
fi

# Set up container-specific overrides
if [ -n "%s" ] && [ -d "/workspace/container-deps" ]; then
    # Install override dependencies (simplified example)
    echo "Setting up container-specific dependencies: %s"
    # In a real implementation, this would run npm/yarn install for those packages
fi
`
	overrideDeps := strings.Join(config.OverrideDeps, " ")
	return fmt.Sprintf(script, overrideDeps, config.DependencyLevel, overrideDeps, overrideDeps, overrideDeps)
}

// setupGitRepository initializes a Git repository in the agent container
func (m *Manager) setupGitRepository(config AgentConfig) error {
	// Prepare clone command with options
	cloneCmd := fmt.Sprintf("git clone %s", config.RepoURL)
	
	// Add branch option if specified
	if config.Branch != "" {
		cloneCmd += fmt.Sprintf(" --branch %s", config.Branch)
	}
	
	// Add depth option if specified
	if config.Depth > 0 {
		cloneCmd += fmt.Sprintf(" --depth %d", config.Depth)
	}
	
	// Add target directory
	cloneCmd += " /workspace/repo"
	
	// Execute clone command
	_, err := m.Exec(config.ID, cloneCmd)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %v", err)
	}
	
	// Apply Git configuration if specified
	if len(config.GitConfig) > 0 {
		for key, value := range config.GitConfig {
			configCmd := fmt.Sprintf("cd /workspace/repo && git config %s \"%s\"", key, value)
			_, err := m.Exec(config.ID, configCmd)
			if err != nil {
				return fmt.Errorf("failed to apply Git config %s: %v", key, err)
			}
		}
	}
	
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

// GetGitStatus retrieves the Git status of the repository in the agent container
func (m *Manager) GetGitStatus(agentID string) (*GitStatus, error) {
	// Get current branch
	branchOutput, err := m.Exec(agentID, "cd /workspace/repo && git branch --show-current")
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %v", err)
	}
	branch := strings.TrimSpace(branchOutput)
	
	// Get current commit
	commitOutput, err := m.Exec(agentID, "cd /workspace/repo && git rev-parse HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %v", err)
	}
	commit := strings.TrimSpace(commitOutput)
	
	// Get modified files
	modifiedOutput, err := m.Exec(agentID, "cd /workspace/repo && git diff --name-only")
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %v", err)
	}
	var modifiedFiles []string
	if modifiedOutput != "" {
		modifiedFiles = strings.Split(strings.TrimSpace(modifiedOutput), "\n")
	}
	
	// Get untracked files
	untrackedOutput, err := m.Exec(agentID, "cd /workspace/repo && git ls-files --others --exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %v", err)
	}
	var untrackedFiles []string
	if untrackedOutput != "" {
		untrackedFiles = strings.Split(strings.TrimSpace(untrackedOutput), "\n")
	}
	
	// Get ahead/behind counts
	aheadBehindOutput, err := m.Exec(agentID, "cd /workspace/repo && git rev-list --count --left-right @{upstream}...HEAD 2>/dev/null || echo '0 0'")
	if err != nil {
		// If error (possibly due to no upstream), default to 0 0
		aheadBehindOutput = "0 0"
	}
	
	aheadBehind := strings.Fields(strings.TrimSpace(aheadBehindOutput))
	ahead, behind := 0, 0
	if len(aheadBehind) >= 2 {
		fmt.Sscanf(aheadBehind[1], "%d", &ahead)
		fmt.Sscanf(aheadBehind[0], "%d", &behind)
	}
	
	return &GitStatus{
		Branch:         branch,
		CurrentCommit:  commit,
		ModifiedFiles:  modifiedFiles,
		UntrackedFiles: untrackedFiles,
		AheadCount:     ahead,
		BehindCount:    behind,
	}, nil
}

// CreateBranch creates a new Git branch in the agent container
func (m *Manager) CreateBranch(agentID, branchName string, checkout bool) error {
	createCmd := fmt.Sprintf("cd /workspace/repo && git branch %s", branchName)
	_, err := m.Exec(agentID, createCmd)
	if err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}
	
	if checkout {
		checkoutCmd := fmt.Sprintf("cd /workspace/repo && git checkout %s", branchName)
		_, err := m.Exec(agentID, checkoutCmd)
		if err != nil {
			return fmt.Errorf("failed to checkout branch: %v", err)
		}
	}
	
	return nil
}

// CheckoutBranch checks out a Git branch in the agent container
func (m *Manager) CheckoutBranch(agentID, branchName string) error {
	checkoutCmd := fmt.Sprintf("cd /workspace/repo && git checkout %s", branchName)
	_, err := m.Exec(agentID, checkoutCmd)
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %v", err)
	}
	
	return nil
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
	timeoutSeconds := int(10)
	if err := m.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeoutSeconds}); err != nil {
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

	// If we get here, need to build the image
	fmt.Printf("Building base image...\n")

	// Create a temporary directory for the Docker build context
	tempDir, err := os.MkdirTemp("", "capsulate-docker-build")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create Dockerfile in temp directory
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	dockerfileContent := `FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    git \
    openssh-client \
    curl \
    build-essential \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set up Git configuration
RUN git config --global init.defaultBranch main

# Create workspace directory
RUN mkdir -p /workspace
WORKDIR /workspace

CMD ["tail", "-f", "/dev/null"]
`
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %v", err)
	}

	// For simplicity, let's use a pull-based approach instead of building
	// This is a workaround since creating a proper tar archive for build context is complex
	fmt.Printf("Using ubuntu image with Git...\n")
	
	// Pull ubuntu image
	out, err := m.dockerClient.ImagePull(ctx, "ubuntu:22.04", types.ImagePullOptions{})
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
			Image: "ubuntu:22.04",
			Cmd:   []string{"/bin/bash", "-c", 
				"apt-get update && apt-get install -y git openssh-client curl build-essential && " +
				"apt-get clean && rm -rf /var/lib/apt/lists/* && " +
				"git config --global init.defaultBranch main && " +
				"mkdir -p /workspace"},
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

	fmt.Printf("Base image built successfully\n")
	return nil
} 