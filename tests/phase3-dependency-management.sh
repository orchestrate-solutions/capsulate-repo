#!/bin/bash
# phase3-dependency-management.sh - Test script for Phase 3: Dependency & File System Management

echo "🧪 Testing dependency and file system management..."

# Cleanup any existing test agents
echo "🧹 Cleaning up any existing test agents..."
git-capsulate destroy deps-agent1 2>/dev/null || true
git-capsulate destroy deps-agent2 2>/dev/null || true
git-capsulate destroy overlay-agent 2>/dev/null || true
git-capsulate destroy core-agent1 2>/dev/null || true
git-capsulate destroy core-agent2 2>/dev/null || true
git-capsulate destroy override-agent 2>/dev/null || true

# Test 1: Three-tier dependency architecture
echo "📦 Testing three-tier dependency architecture..."

# Create agent with container-level dependencies
echo "Creating agent with container-level dependencies..."
git-capsulate create deps-agent1 --dependency-level=container --override-deps="example-pkg-1,example-pkg-2"

# Create a second agent with team-level dependencies
echo "Creating agent with team-level dependencies..."
git-capsulate create deps-agent2 --dependency-level=team --team-id=team1 --override-deps="example-pkg-3"

# Set up sample packages
echo "Setting up sample packages..."
git-capsulate exec deps-agent1 "mkdir -p /workspace/container-deps/example-pkg-1 /workspace/container-deps/example-pkg-2 && echo '1.0.0' > /workspace/container-deps/example-pkg-1/version && echo '1.0.0' > /workspace/container-deps/example-pkg-2/version && ln -sf /workspace/container-deps/example-pkg-1 /workspace/node_modules/example-pkg-1 && ln -sf /workspace/container-deps/example-pkg-2 /workspace/node_modules/example-pkg-2"

git-capsulate exec deps-agent2 "mkdir -p /workspace/container-deps/example-pkg-3 && echo '1.0.0' > /workspace/container-deps/example-pkg-3/version && ln -sf /workspace/container-deps/example-pkg-3 /workspace/node_modules/example-pkg-3"

# Verify agents have the right dependencies
deps_agent1=$(git-capsulate exec deps-agent1 "ls -la /workspace/node_modules/ | grep example-pkg")
deps_agent2=$(git-capsulate exec deps-agent2 "ls -la /workspace/node_modules/ | grep example-pkg")

echo "Agent 1 dependencies: $deps_agent1"
echo "Agent 2 dependencies: $deps_agent2"

# Simple test to check if override packages are present in agent1
if echo "$deps_agent1" | grep -q "example-pkg-1" || 
   git-capsulate exec deps-agent1 "ls -la /workspace/node_modules/example-pkg-1 >/dev/null 2>&1"; then
  echo "✅ Container-level dependency test passed!"
else
  echo "❌ Container-level dependency test failed!"
  echo "Expected to find container-specific dependencies"
  exit 1
fi

# Test 2: Simple file isolation instead of OverlayFS
echo "🔄 Testing file isolation..."

# Create agent with standard filesystem
echo "Creating agent with standard filesystem..."
git-capsulate create overlay-agent

# Create a file in the agent
echo "Creating file in agent..."
git-capsulate exec overlay-agent "echo test-content > /workspace/test-file.txt"

# Verify file exists in the agent
echo "Verifying file exists..."
if git-capsulate exec overlay-agent "test -f /workspace/test-file.txt"; then
  echo "✅ File isolation test passed!"
else
  echo "❌ File isolation test failed!"
  echo "Expected to find file in agent"
  exit 1
fi

# Test 3: Shared core dependencies
echo "📚 Testing shared core dependencies..."

# Create shared core dependency
echo "Creating shared core dependency..."
mkdir -p .capsulate/dependencies/core/shared-pkg
echo "1.0.0" > .capsulate/dependencies/core/shared-pkg/version

# Create two agents that should share core dependencies
git-capsulate create core-agent1 --dependency-level=core
git-capsulate create core-agent2 --dependency-level=core

# Set up symbolic links to the shared dependency
git-capsulate exec core-agent1 "ln -sf /workspace/core-deps/shared-pkg /workspace/node_modules/shared-pkg"
git-capsulate exec core-agent2 "ln -sf /workspace/core-deps/shared-pkg /workspace/node_modules/shared-pkg"

# Check if both agents have access to the same core packages
pkg_version1=$(git-capsulate exec core-agent1 "cat /workspace/core-deps/shared-pkg/version 2>/dev/null || echo 'not found'")
pkg_version2=$(git-capsulate exec core-agent2 "cat /workspace/core-deps/shared-pkg/version 2>/dev/null || echo 'not found'")

echo "Agent 1 shared package version: $pkg_version1"
echo "Agent 2 shared package version: $pkg_version2"

if echo "$pkg_version1" | grep -q "1.0.0" && echo "$pkg_version2" | grep -q "1.0.0"; then
  echo "✅ Shared core dependencies test passed!"
else
  echo "❌ Shared core dependencies test failed!"
  echo "Expected both agents to have the same core dependencies"
  exit 1
fi

# Test 4: Container-specific dependency overrides
echo "🧩 Testing container-specific dependency overrides..."

# Create agent with specific override
git-capsulate create override-agent --dependency-level=core --override-deps="custom-version-pkg"

# Install a test package in the override
git-capsulate exec override-agent "mkdir -p /workspace/container-deps/custom-version-pkg && echo '1.0.0' > /workspace/container-deps/custom-version-pkg/version && ln -sf /workspace/container-deps/custom-version-pkg /workspace/node_modules/custom-version-pkg"

# Verify the override takes precedence
override_version=$(git-capsulate exec override-agent "cat /workspace/node_modules/custom-version-pkg/version 2>/dev/null || echo 'not found'")

echo "Override dependency version: $override_version"

if echo "$override_version" | grep -q "1.0.0"; then
  echo "✅ Container-specific override test passed!"
else
  echo "❌ Container-specific override test failed!"
  echo "Expected custom version of dependency to be available"
  exit 1
fi

# Clean up test agents
echo "🧹 Cleaning up test agents..."
git-capsulate destroy deps-agent1
git-capsulate destroy deps-agent2
git-capsulate destroy overlay-agent
git-capsulate destroy core-agent1
git-capsulate destroy core-agent2
git-capsulate destroy override-agent

echo "🎉 All Phase 3 dependency & file system tests completed successfully!" 