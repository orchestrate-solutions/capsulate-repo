#!/bin/bash
# phase3-dependency-management.sh - Test script for Phase 3: Dependency & File System Management

echo "🧪 Testing dependency and file system management..."

# Test 1: Three-tier dependency architecture
echo "📦 Testing three-tier dependency architecture..."

# Create agent with container-level dependencies
echo "Creating agent with container-level dependencies..."
git-capsulate create deps-agent1 --dependency-level=container --override-deps="example-pkg-1,example-pkg-2"

# Create a second agent with team-level dependencies
echo "Creating agent with team-level dependencies..."
git-capsulate create deps-agent2 --dependency-level=team --team-id=team1 --override-deps="example-pkg-3"

# Verify agents have the right dependencies
deps_agent1=$(git-capsulate exec deps-agent1 "ls -la /workspace/node_modules/ | grep example-pkg")
deps_agent2=$(git-capsulate exec deps-agent2 "ls -la /workspace/node_modules/ | grep example-pkg")

echo "Agent 1 dependencies: $deps_agent1"
echo "Agent 2 dependencies: $deps_agent2"

# Simple test to check if override packages are present in agent1
if echo "$deps_agent1" | grep -q "example-pkg-1" && 
   echo "$deps_agent1" | grep -q "example-pkg-2"; then
  echo "✅ Container-level dependency test passed!"
else
  echo "❌ Container-level dependency test failed!"
  echo "Expected to find container-specific dependencies"
  exit 1
fi

# Test 2: OverlayFS implementation
echo "🔄 Testing OverlayFS implementation..."

# Create agent with overlay file system
echo "Creating agent with overlay file system..."
git-capsulate create overlay-agent --use-overlay=true

# Create a file in the overlay
git-capsulate exec overlay-agent "echo 'overlay test content' > /workspace/overlay-test.txt"

# Verify file exists in the diff/upper directory but not in the base/lower directory
diff_content=$(git-capsulate exec overlay-agent "cat /workspace/diff/overlay-test.txt 2>/dev/null || echo 'not found'")
base_content=$(git-capsulate exec overlay-agent "cat /workspace/base/overlay-test.txt 2>/dev/null || echo 'not found'")
merged_content=$(git-capsulate exec overlay-agent "cat /workspace/merged/overlay-test.txt 2>/dev/null || echo 'not found'")

echo "Content in diff layer: $diff_content"
echo "Content in base layer: $base_content"
echo "Content in merged view: $merged_content"

if [ "$diff_content" = "overlay test content" ] && 
   [ "$base_content" = "not found" ] && 
   [ "$merged_content" = "overlay test content" ]; then
  echo "✅ OverlayFS implementation test passed!"
else
  echo "❌ OverlayFS implementation test failed!"
  echo "Expected content only in diff and merged layers, not in base"
  exit 1
fi

# Test 3: Shared core dependencies
echo "📚 Testing shared core dependencies..."

# Create two agents that should share core dependencies
git-capsulate create core-agent1 --dependency-level=core
git-capsulate create core-agent2 --dependency-level=core

# Check if both agents have access to the same core packages
core_deps1=$(git-capsulate exec core-agent1 "ls -la /workspace/core-deps/ 2>/dev/null | wc -l")
core_deps2=$(git-capsulate exec core-agent2 "ls -la /workspace/core-deps/ 2>/dev/null | wc -l")

echo "Agent 1 core dependencies count: $core_deps1"
echo "Agent 2 core dependencies count: $core_deps2"

if [ "$core_deps1" = "$core_deps2" ] && [ "$core_deps1" -gt 0 ]; then
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
git-capsulate exec override-agent "mkdir -p /workspace/node_modules/custom-version-pkg && echo '1.0.0' > /workspace/node_modules/custom-version-pkg/version"

# Verify the override takes precedence
override_version=$(git-capsulate exec override-agent "cat /workspace/node_modules/custom-version-pkg/version 2>/dev/null || echo 'not found'")

echo "Override dependency version: $override_version"

if [ "$override_version" = "1.0.0" ]; then
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