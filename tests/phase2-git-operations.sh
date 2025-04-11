#!/bin/bash
# phase2-git-operations.sh - Test script for Phase 2: Git Operations & Branch Management

echo "🧪 Testing Git operations within isolated environments..."

# Set up test agents
echo "📦 Creating test agents..."
git-capsulate create test-agent1
git-capsulate create test-agent2

# Test 1: Branch creation and isolation
echo "🔄 Testing branch creation and isolation..."
git-capsulate exec test-agent1 "git checkout -b feature-branch-1"
git-capsulate exec test-agent1 "echo 'Feature 1 content' > feature.txt"
git-capsulate exec test-agent1 "git add feature.txt"
git-capsulate exec test-agent1 "git commit -m 'Add feature 1 content'"

git-capsulate exec test-agent2 "git checkout -b feature-branch-2"
git-capsulate exec test-agent2 "echo 'Feature 2 content' > feature.txt"
git-capsulate exec test-agent2 "git add feature.txt"
git-capsulate exec test-agent2 "git commit -m 'Add feature 2 content'"

# Verify branches are isolated
agent1_branch=$(git-capsulate exec test-agent1 "git branch --show-current")
agent2_branch=$(git-capsulate exec test-agent2 "git branch --show-current")
agent1_content=$(git-capsulate exec test-agent1 "cat feature.txt")
agent2_content=$(git-capsulate exec test-agent2 "cat feature.txt")

echo "Agent 1 branch: $agent1_branch, content: $agent1_content"
echo "Agent 2 branch: $agent2_branch, content: $agent2_content"

if [ "$agent1_branch" == "feature-branch-1" ] && 
   [ "$agent2_branch" == "feature-branch-2" ] &&
   [ "$agent1_content" == "Feature 1 content" ] && 
   [ "$agent2_content" == "Feature 2 content" ]; then
  echo "✅ Branch isolation test passed!"
else
  echo "❌ Branch isolation test failed!"
  exit 1
fi

# Test 2: Branch switching
echo "🔄 Testing branch switching..."
git-capsulate exec test-agent1 "git checkout -b another-branch"
git-capsulate exec test-agent1 "echo 'Another branch content' > another.txt"
git-capsulate exec test-agent1 "git add another.txt"
git-capsulate exec test-agent1 "git commit -m 'Add another content'"

# Switch back to first branch
git-capsulate exec test-agent1 "git checkout feature-branch-1"
current_branch=$(git-capsulate exec test-agent1 "git branch --show-current")
file_exists=$(git-capsulate exec test-agent1 "[ -f another.txt ] && echo 'exists' || echo 'not exists'")

if [ "$current_branch" == "feature-branch-1" ] && [ "$file_exists" == "not exists" ]; then
  echo "✅ Branch switching test passed!"
else
  echo "❌ Branch switching test failed!"
  exit 1
fi

# Test 3: Git status visibility
echo "🔍 Testing Git status visibility..."
git-capsulate exec test-agent1 "echo 'New content' >> feature.txt"
status_output=$(git-capsulate status test-agent1)

if [[ "$status_output" == *"feature-branch-1"* ]] && [[ "$status_output" == *"modified"* ]]; then
  echo "✅ Git status visibility test passed!"
else
  echo "❌ Git status visibility test failed!"
  exit 1
fi

# Test 4: Configurable repository cloning
echo "📦 Testing configurable repository cloning..."
# Create a temporary test repo
mkdir -p /tmp/test-repo
cd /tmp/test-repo
git init
echo "Test repo content" > README.md
git add README.md
git commit -m "Initial commit"
cd -

# Clone with custom options
git-capsulate create clone-agent --repo=/tmp/test-repo --branch=master --depth=1
readme_content=$(git-capsulate exec clone-agent "cat README.md")

if [ "$readme_content" == "Test repo content" ]; then
  echo "✅ Repository cloning test passed!"
else
  echo "❌ Repository cloning test failed!"
  exit 1
fi

# Clean up test agents
echo "🧹 Cleaning up test agents..."
git-capsulate destroy test-agent1
git-capsulate destroy test-agent2
git-capsulate destroy clone-agent

echo "🎉 All Phase 2 Git operations tests completed successfully!" 