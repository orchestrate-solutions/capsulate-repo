# 🔒 CapsulateRepo

A tool for creating isolated Git environments in Docker containers to enable parallel features, experiments, and fixes without cluttering your main workspace.

## 📋 Overview

CapsulateRepo provides containerized Git environments with proper isolation. It allows developers to work on multiple isolated branches simultaneously without the risk of accidental changes leaking between branches. The isolation is achieved through Docker containers, each with its own Git state.

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