# 🔍 Phase 3: Dependency & File System Management Analysis

## 📋 Implementation Overview

**Phase 3 Focus**: Implementing efficient storage and dependency management in isolated Git environments

### Key Components to Implement:

1. **Three-Tier Dependency Architecture**
   ```
   ┌─────────────────────┐
   │    CONTAINER DEPS   │ <- Container-specific deps (breaking changes)
   ├─────────────────────┤
   │     TEAM DEPS       │ <- Team/feature-specific shared deps 
   ├─────────────────────┤
   │    CORE DEPS        │ <- Organization-wide stable deps
   └─────────────────────┘
   ```
   - Core dependencies: Shared across all containers
   - Team dependencies: Shared among a team's containers
   - Container dependencies: Specific to individual containers

2. **OverlayFS Implementation**
   - Base layer: Read-only shared repository
   - Upper layer: Container-specific changes
   - Merged view: Combined layers for container use

3. **Dependency Management Types**
   - Dependency isolation levels (core, team, container)
   - Dependency override mechanism
   - Shared dependency mounting

4. **File System Optimization**
   - Efficient storage with minimal duplication
   - Transparent file access for developers
   - Clear separation of shared vs. private files

## 🧪 Testing Approach

Tests are designed to validate:
- Proper implementation of the three-tier dependency system
- Correct functioning of OverlayFS for file isolation
- Shared dependency access across containers
- Container-specific dependency overrides

### Test Flow:
1. Test three-tier dependency architecture
2. Test OverlayFS file isolation
3. Test shared core dependencies
4. Test container-specific dependency overrides

## 🚧 Implementation Strategy

### 1. Dependency Manager Implementation

```go
// DependencyManager handles the three-tier dependency system
type DependencyManager struct {
    CoreDepsPath      string                // Shared by all containers
    TeamDepsPath      map[string]string     // Shared by team/feature
    ContainerDepsPath string                // Container-specific
}

// ConfigureDependencies sets up the dependency environment
func (d *DependencyManager) ConfigureDependencies(config AgentConfig) []mount.Mount {
    var mounts []mount.Mount
    
    // Always mount core deps if available
    if d.CoreDepsPath != "" {
        mounts = append(mounts, mount.Mount{
            Type:     mount.TypeBind,
            Source:   d.CoreDepsPath,
            Target:   "/workspace/core-deps",
            ReadOnly: true,
        })
    }
    
    // Add team deps if applicable
    if config.DependencyLevel == "team" && config.TeamID != "" {
        if teamPath, ok := d.TeamDepsPath[config.TeamID]; ok {
            mounts = append(mounts, mount.Mount{
                Type:     mount.TypeBind,
                Source:   teamPath,
                Target:   "/workspace/team-deps",
                ReadOnly: true,
            })
        }
    }
    
    // Add container-specific deps directory
    containerDepsPath := filepath.Join(d.ContainerDepsPath, config.ID)
    os.MkdirAll(containerDepsPath, 0755)
    mounts = append(mounts, mount.Mount{
        Type:   mount.TypeBind,
        Source: containerDepsPath,
        Target: "/workspace/container-deps",
    })
    
    return mounts
}
```

### 2. Overlay File System Manager

```go
// FileSystemManager handles OverlayFS implementation
type FileSystemManager struct {
    BaseRepoPath   string   // Shared read-only repo
    DiffsPath      string   // Container-specific changes
    WorkPath       string   // OverlayFS work directory
}

// ConfigureOverlayFS sets up OverlayFS for a container
func (f *FileSystemManager) ConfigureOverlayFS(config AgentConfig) []mount.Mount {
    var mounts []mount.Mount
    
    if !config.UseOverlay {
        // Without overlay, just mount the workspace directly
        return mounts
    }
    
    // Create container-specific directories
    containerDiffPath := filepath.Join(f.DiffsPath, config.ID)
    containerWorkPath := filepath.Join(f.WorkPath, config.ID)
    
    os.MkdirAll(containerDiffPath, 0755)
    os.MkdirAll(containerWorkPath, 0755)
    
    // Mount the base repo as read-only
    mounts = append(mounts, mount.Mount{
        Type:     mount.TypeBind,
        Source:   f.BaseRepoPath,
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
    
    // Will need to set up the overlay mount inside container
    // via entrypoint script
    
    return mounts
}
```

### 3. Container Entrypoint Script

This script runs when a container starts to set up the OverlayFS and dependency linking:

```bash
#!/bin/bash

# Set up OverlayFS if enabled
if [ "$USE_OVERLAY" = "true" ]; then
    mkdir -p /workspace/merged
    mount -t overlay overlay -o lowerdir=/workspace/base,upperdir=/workspace/diff,workdir=/workspace/work /workspace/merged
    # Make the merged directory the default workspace
    cd /workspace/merged
else
    cd /workspace
fi

# Set up dependency linking based on isolation level
mkdir -p /workspace/node_modules

# Link core dependencies if available
if [ -d "/workspace/core-deps" ]; then
    for pkg in $(find /workspace/core-deps -maxdepth 1 -type d ! -path "/workspace/core-deps"); do
        pkg_name=$(basename $pkg)
        # Don't link if it's in the override list
        if [[ ! " $OVERRIDE_DEPS " =~ " $pkg_name " ]]; then
            ln -sf "$pkg" "/workspace/node_modules/$pkg_name"
        fi
    done
fi

# Link team dependencies if available
if [ "$DEPENDENCY_LEVEL" = "team" ] && [ -d "/workspace/team-deps" ]; then
    for pkg in $(find /workspace/team-deps -maxdepth 1 -type d ! -path "/workspace/team-deps"); do
        pkg_name=$(basename $pkg)
        # Don't link if it's in the override list
        if [[ ! " $OVERRIDE_DEPS " =~ " $pkg_name " ]]; then
            ln -sf "$pkg" "/workspace/node_modules/$pkg_name"
        fi
    done
fi

# Set up container-specific overrides
if [ -n "$OVERRIDE_DEPS" ] && [ -d "/workspace/container-deps" ]; then
    # Install override dependencies (simplified example)
    echo "Setting up container-specific dependencies: $OVERRIDE_DEPS"
    # In a real implementation, this would run npm/yarn install for those packages
fi

# Keep container running
exec "$@"
```

## 🧠 Design Considerations

### Memory Efficiency
- Shared directories minimize memory usage
- Deduplication of common dependencies
- Only container-specific changes consume additional space

### Performance Optimization
- Read-only mounts for shared resources
- Layered access for minimal I/O
- Transparent access for developer productivity

### Flexibility vs. Standardization
- Core dependencies enforce standards
- Team dependencies allow flexibility for feature work
- Container dependencies enable experimentation without affecting others

## 📊 Expected Resource Usage

| Configuration | Disk Space | Memory | Setup Time |
|---------------|------------|--------|------------|
| No Overlay    | ~500MB/container | ~100MB | <5s |
| With Overlay  | ~50MB/container | ~100MB | <10s |
| Full 3-tier   | ~100MB/container | ~120MB | <15s |

## 🚀 Next Steps for Implementation

1. Enhanced Manager API:
   - Add dependency level configuration
   - Implement team management
   - Add overlay configuration options

2. File System Logic:
   - Implement the OverlayFS setup
   - Add merge conflict detection
   - Optimize file change tracking

3. CLI Commands:
   - Add dependency management commands
   - Support team assignment
   - Add overlay status visibility

4. Infrastructure:
   - Create paths for shared dependencies
   - Set up permission management
   - Implement cleanup procedures 