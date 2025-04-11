#!/bin/bash
# Phase 1 - Core Infrastructure Tests
# Tests the basic container isolation and SSH authentication functionality

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🧪 Phase 1: Testing Core Infrastructure"

# Test 1: Container Creation
echo "📋 Test 1.1: Container Creation"
./git-isolate create test-agent1

# Verify container exists and is running
if docker ps | grep -q "git-isolate-test-agent1"; then
  echo "✅ Container creation test passed!"
else
  echo "❌ Container creation test failed!"
  exit 1
fi

# Test 2: Basic Command Execution
echo "📋 Test 1.2: Command Execution"
result=$(./git-isolate exec test-agent1 "echo 'hello from container'")

if [[ "$result" == *"hello from container"* ]]; then
  echo "✅ Command execution test passed!"
else
  echo "❌ Command execution test failed!"
  exit 1
fi

# Test 3: SSH Key Sharing
echo "📋 Test 1.3: SSH Authentication"
# Setup a mock git repo that requires SSH for this test
setup_mock_repo() {
  mkdir -p /tmp/mock-git-repo
  cd /tmp/mock-git-repo
  git init
  echo "test content" > test-file.txt
  git add test-file.txt
  git commit -m "Initial commit"
  cd -
}

# Test SSH key mounting by trying a git operation that would require authentication
setup_mock_repo
ssh_test=$(./git-isolate exec test-agent1 "GIT_SSH_COMMAND='ssh -o StrictHostKeyChecking=no' git ls-remote git@github.com:octocat/Hello-World.git 2>&1 || echo 'SSH Failed'")

if [[ "$ssh_test" != *"Permission denied"* ]] && [[ "$ssh_test" != *"SSH Failed"* ]]; then
  echo "✅ SSH authentication test passed!"
else
  echo "❌ SSH authentication test failed! Output: $ssh_test"
  exit 1
fi

# Test 4: Container Destruction
echo "📋 Test 1.4: Container Destruction"
./git-isolate destroy test-agent1

if ! docker ps | grep -q "git-isolate-test-agent1"; then
  echo "✅ Container destruction test passed!"
else
  echo "❌ Container destruction test failed!"
  exit 1
fi

echo "✅ All Phase 1 tests passed!" 