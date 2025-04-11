# 🔒 CapsulateRepo

A tool for creating isolated Git environments in Docker containers to enable parallel features, experiments, and fixes without cluttering your main workspace.

## 📋 Overview

CapsulateRepo provides containerized Git environments with proper isolation. It allows developers to work on multiple isolated branches simultaneously without the risk of accidental changes leaking between branches. The isolation is achieved through Docker containers, each with its own Git state.

## 🔄 How It Works

### OverlayFS: Efficient File System Isolation

CapsulateRepo uses OverlayFS to create efficient, isolated environments without duplicating files:

```
┌─────────────────────────────────────┐
│           CONTAINER VIEW            │ <- What you see when working
├─────────────────────────────────────┤
│                                     │
│  ┌─────────────────────────────┐    │
│  │      Your Changes (Diff)    │    │ <- Only your modifications 
│  ├─────────────────────────────┤    │    are stored here
│  │                             │    │
│  │  ┌─────────────────────┐    │    │
│  │  │  Base Repository    │    │    │ <- Read-only, shared across
│  │  │  (Read-only)        │    │    │    all containers
│  │  └─────────────────────┘    │    │
│  │                             │    │
│  └─────────────────────────────┘    │
│                                     │
└─────────────────────────────────────┘
```

When you open a file, you see a merged view that combines:
1. The original file from the base repository (bottom layer)
2. Any changes you've made (upper layer)

Changes you make are only stored in the diff layer, while the base repository remains untouched. This provides several benefits:

### Three-Tier Dependency Management

CapsulateRepo implements a sophisticated dependency management system that balances standardization with flexibility:

```
┌─────────────────────────────────────┐
│       CONTAINER DEPENDENCIES        │ <- Container-specific deps
├─────────────────────────────────────┤    (for experimentation)
│                                     │
│  ┌─────────────────────────────┐    │
│  │      TEAM DEPENDENCIES      │    │ <- Team/feature-specific deps
│  ├─────────────────────────────┤    │    (shared among a team)
│  │                             │    │
│  │  ┌─────────────────────┐    │    │
│  │  │   CORE DEPENDENCIES │    │    │ <- Organization-wide deps
│  │  │                     │    │    │    (shared by all containers)
│  │  └─────────────────────┘    │    │
│  │                             │    │
│  └─────────────────────────────┘    │
│                                     │
└─────────────────────────────────────┘
```

**How it works:**

1. **Core Dependencies**: Shared across all containers
   - Ensures standardization across the organization
   - Reduces duplication and saves storage
   - Examples: foundational libraries, testing frameworks, core utilities

2. **Team Dependencies**: Shared within specific teams or features
   - Balances standardization with team-specific needs
   - Enables team autonomy while maintaining consistency
   - Examples: UI frameworks for frontend teams, data processing libraries for backend teams

3. **Container Dependencies**: Specific to individual containers
   - Allows full experimentation freedom
   - Can override or add to team/core dependencies
   - Perfect for testing new libraries, version upgrades, or experimental features

This approach gives you the perfect balance between standardization, efficiency, and flexibility - critical for both human and AI-driven development workflows.

### Why This Matters for Human-in-the-Loop Development

1. **Parallel experimentation**: Multiple AI agents or humans can work on the same codebase simultaneously without interference
   
2. **Efficient storage**: Only store the changes, not entire copies of repositories

3. **Safe isolation**: Changes in one environment never leak into another

4. **Visibility**: External tools like VS Code can seamlessly work with these environments by connecting to the workspace directory

5. **Context switching**: Instantly switch between different isolated environments without the overhead of git stashing or branch switching

6. **Dependency isolation**: Each environment can have its own dependencies without conflicts

This architecture is particularly powerful for human-in-the-loop development where:
- AI agents can suggest changes in isolated environments
- Humans can review and modify those changes
- Multiple experiments can run concurrently
- Teams can collaborate without stepping on each other's work

### Accessing the Environment

External tools can access these environments by:
1. Opening the filesystem at `<workspace-dir>/.capsulate/workspaces/<agent-id>`
2. Using the `git-capsulate exec` command to run operations inside the container
3. Using VS Code's Remote Container extension to connect directly to the container

### Human-AI Collaboration Workflows

CapsulateRepo enables powerful workflows between humans and AI agents:

```
┌───────────────────────────────────────────────────────────────────┐
│                       COLLABORATIVE WORKFLOW                       │
└───────────────────────────────────────────────────────────────────┘
                                │
         ┌────────────────────┬─┴───────────────┬─────────────────┐
         │                    │                 │                 │
┌────────▼───────┐   ┌────────▼───────┐ ┌───────▼────────┐ ┌─────▼─────────┐
│   AI Agent 1   │   │   AI Agent 2   │ │   AI Agent 3   │ │  Human Dev    │
│  Environment   │   │  Environment   │ │  Environment   │ │  Environment  │
└────────┬───────┘   └────────┬───────┘ └───────┬────────┘ └─────┬─────────┘
         │                    │                 │                 │
         │                    │                 │                 │
┌────────▼────────────────────▼─────────────────▼─────────────────▼──────────┐
│                                                                             │
│                            BASE REPOSITORY                                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Example workflows:**

1. **Parallel Feature Development**
   - Multiple AI agents work on different features in isolated environments
   - Human reviews and refines each feature without context switching

2. **Experimental Variations**
   - AI generates multiple approaches to solving the same problem
   - Each variation is in its own isolated environment
   - Human can review and compare without messy branch switching

3. **Code Review Pipeline**
   - AI agent 1 generates code
   - AI agent 2 reviews and suggests improvements
   - AI agent 3 writes tests
   - Human makes final decisions and merges

4. **Dependency Experiments**
   - Test different dependency versions in parallel
   - Evaluate breaking changes safely
   - Compare performance between versions

This architecture makes CapsulateRepo ideal for orchestrating complex workflows between humans and AI agents, enabling truly parallel development.

## 🛠️ Implementation Progress

### Phase 1: Core Infrastructure ✅
- Container creation and management
- SSH authentication sharing
- Basic command execution
- Container lifecycle management

### Phase 2: Git Operations & Branch Management ✅
- Git repository cloning
- Branch creation and management
- Status tracking and visualization
- Repository sharing between containers

### Phase 3: Dependency & File System Management ✅
- Three-tier dependency management (core, team, container levels)
- Efficient file storage with OverlayFS
- Dependency isolation and overrides
- Team-based dependency sharing

### Phase 4: Synchronization & Scaling ⏳
- Background syncing from central branches
- Conflict detection and management
- Scaling to many containers efficiently

### Phase 5: Monitoring & Management ⏳
- Resource usage monitoring
- Container health checks
- Branch activity metrics

## 🧪 Testing

Each phase includes comprehensive tests that validate the implemented functionality:

- `tests/phase1-core-infrastructure.sh`: Tests for container creation, destruction, and command execution
- `tests/phase2-git-operations.sh`: Tests for Git isolation, branch management, and status reporting
- `tests/phase3-dependency-management.sh`: Tests for dependency management and OverlayFS functionality

## 📚 Documentation

Detailed analysis documents for each phase are available in the `docs/` directory:

- `docs/phase1-analysis.md`: Core infrastructure design and implementation
- `docs/phase2-analysis.md`: Git operations and branch management implementation
- `docs/phase3-analysis.md`: Dependency and file system management architecture

## 🚀 Usage

### Create an isolated Git environment

```bash
git-capsulate create my-feature --repo=git@github.com:user/repo.git --branch=main --dependency-level=team --team-id=frontend --use-overlay=true
```

### Execute commands in the environment

```bash
git-capsulate exec my-feature "git status"
```

### Create and checkout branches

```bash
git-capsulate branch my-feature new-branch -c
git-capsulate checkout my-feature main
```

### Manage dependencies

```bash
git-capsulate add-dep my-feature lodash
git-capsulate list-deps my-feature
```

### Work with teams and shared dependencies

```bash
git-capsulate create-team frontend
git-capsulate add-team-dep frontend react
```

### Check overlay filesystem status

```bash
git-capsulate overlay-status my-feature
```

### Destroy the environment

```bash
git-capsulate destroy my-feature
```

## 📋 Requirements

- Docker installed and running
- Go 1.21+ (for building from source)
- Git
- SSH keys configured for Git operations

## 🔧 Installation

1. Clone this repository:
   ```bash
   git clone https://github.com/your-org/CapsulateRepo.git
   cd CapsulateRepo
   ```

2. Make the script executable:
   ```bash
   chmod +x git-capsulate
   ```

3. (Optional) Add to your PATH for global access:
   ```bash
   export PATH="$PATH:$(pwd)"
   ```

## 🚀 Quick Start

### Create an isolated Git environment

```bash
./git-capsulate create my-feature
```

### Execute commands in the isolated environment

```bash
./git-capsulate exec my-feature "git clone git@github.com:your-org/your-repo.git repo"
./git-capsulate exec my-feature "cd repo && git checkout -b feature/my-feature"
./git-capsulate exec my-feature "cd repo && echo 'New feature code' > feature.txt"
./git-capsulate exec my-feature "cd repo && git add feature.txt && git commit -m 'Add new feature'"
```

### Destroy the isolated environment

```bash
./git-capsulate destroy my-feature
```

## 🔧 Advanced Usage

### Create with dependency isolation

```bash
./git-capsulate create deps-test --dependency-level=container --override-deps="lodash,express"
```

### Create with overlay filesystem

```bash
./git-capsulate create overlay-test --use-overlay=true
```

## 🤝 Contributing

1. Make sure tests pass for your changes
2. Follow Go coding conventions
3. Add tests for new functionality
4. Update documentation as needed

## 📃 License

MIT