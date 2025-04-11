#!/bin/bash
# Phase 1 - Core Infrastructure Tests
# Tests the basic container isolation and SSH authentication functionality
# Using temporary Bash implementation

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🧪 Phase 1: Testing Core Infrastructure"

# Test 1: Container Creation
echo "📋 Test 1.1: Container Creation"
./git-capsulate-test.sh create test-agent1

# Verify container exists and is running
if docker ps | grep -q "capsulate-test-agent1"; then
  echo "✅ Container creation test passed!"
else
  echo "❌ Container creation test failed!"
  exit 1
fi

# Test 2: Basic Command Execution
echo "📋 Test 1.2: Command Execution"
result=$(./git-capsulate-test.sh exec test-agent1 "echo 'hello from container'")

if [[ "$result" == *"hello from container"* ]]; then
  echo "✅ Command execution test passed!"
else
  echo "❌ Command execution test failed!"
  exit 1
fi

# Test 3: SSH Key Sharing
echo "📋 Test 1.3: SSH Authentication"
# Test SSH key mounting by checking if the directory is mounted
ssh_test=$(./git-capsulate-test.sh exec test-agent1 "mount | grep '/root/.ssh'")

if [[ "$ssh_test" == *"/root/.ssh"* ]]; then
  echo "✅ SSH key mounting test passed!"
else
  echo "❌ SSH key mounting test failed! Output: $ssh_test"
  exit 1
fi

# Test 4: Container Destruction
echo "📋 Test 1.4: Container Destruction"
./git-capsulate-test.sh destroy test-agent1

if ! docker ps | grep -q "capsulate-test-agent1"; then
  echo "✅ Container destruction test passed!"
else
  echo "❌ Container destruction test failed!"
  exit 1
fi

echo "✅ All Phase 1 tests passed!" 