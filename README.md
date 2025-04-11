# 🚀 CapsulateRepo

A tool for managing isolated Git environments using Docker containers, allowing multiple parallel development streams on the same codebase.

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

## 🧪 Running Tests

We use test-driven development. To run the tests:

```bash
cd tests
./phase1-core-infrastructure.sh
```

## 📚 Architecture

This tool follows a layered architecture:

1. **CLI Layer** - Command-line interface using Cobra
2. **Agent Manager** - Manages lifecycle of isolated environments
3. **Docker Integration** - Handles container operations
4. **Git Operations** - Manages Git operations in containers

## 🤝 Contributing

1. Make sure tests pass for your changes
2. Follow Go coding conventions
3. Add tests for new functionality
4. Update documentation as needed

## 📃 License

MIT 