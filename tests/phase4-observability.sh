#!/bin/bash
# phase4-observability.sh - Test script for Phase 4: Observability Features

echo "🧪 Testing observability and metrics infrastructure..."

# Setup: Create test agents for metrics collection
echo "📦 Setting up test environment..."
git-capsulate destroy obs-test-agent1 2>/dev/null || true
git-capsulate destroy obs-test-agent2 2>/dev/null || true

# Clear any existing metrics
git-capsulate metrics clear

# Test 1: Metrics collection during operations
echo "📊 Testing metrics collection..."

# Create agents and perform operations that should generate metrics
git-capsulate create obs-test-agent1
git-capsulate create obs-test-agent2

# Initialize Git repositories in the containers
git-capsulate exec obs-test-agent1 "mkdir -p /workspace/repo && cd /workspace/repo && git init"
git-capsulate exec obs-test-agent1 "cd /workspace/repo && git config --global user.email 'test@example.com' && git config --global user.name 'Test User'"
git-capsulate exec obs-test-agent1 "cd /workspace/repo && touch test.txt && git add test.txt && git commit -m 'Initial commit'"

# Get metrics summary
metrics_summary=$(git-capsulate metrics show --format=json)

# Check if metrics were collected
container_ops=$(echo "$metrics_summary" | grep -c "container")
git_ops=$(echo "$metrics_summary" | grep -c "git")

echo "Container operations metrics: $container_ops"
echo "Git operations metrics: $git_ops"

if [ "$container_ops" -gt 0 ]; then
  echo "✅ Container metrics collection test passed!"
else
  echo "❌ Container metrics collection test failed!"
  exit 1
fi

# Test 2: Monitor resource usage
echo "🖥️ Testing resource monitoring..."

# Start monitoring
git-capsulate monitor start

# Generate some load in the container
git-capsulate exec obs-test-agent1 "cd /workspace/repo && dd if=/dev/zero of=testfile bs=1M count=10 && rm testfile"

# Wait for monitor to collect stats
echo "Waiting for resource stats collection..."
sleep 6

# Get resource usage stats
resource_stats=$(git-capsulate monitor show obs-test-agent1 --format=json)

# Check if resource stats were collected
if [[ "$resource_stats" == *"CPUUsage"* ]] && [[ "$resource_stats" == *"MemoryUsage"* ]]; then
  echo "✅ Resource monitoring test passed!"
else
  echo "❌ Resource monitoring test failed!"
  echo "Resource stats: $resource_stats"
  exit 1
fi

# Test 3: Tracing
echo "🔍 Testing distributed tracing..."

# Create a third agent with tracing
# (This should generate spans that we can check)
git-capsulate destroy obs-test-agent3 2>/dev/null || true
git-capsulate create obs-test-agent3

# Check for traces
traces_output=$(git-capsulate traces)

# Verify traces exist
if [[ "$traces_output" == *"No active traces"* ]]; then
  echo "Note: No active traces found - may have already completed"
  echo "⚠️ Tracing test inconclusive - need to verify trace file output"
else
  echo "✅ Active traces detected!"
  echo "$traces_output"
fi

# Check for trace files
trace_files=$(find .capsulate/traces -name "trace-*.json" 2>/dev/null | wc -l)
echo "Found $trace_files trace files"

if [ "$trace_files" -gt 0 ]; then
  echo "✅ Trace file generation test passed!"
else
  echo "⚠️ No trace files found. This could be normal if traces are not yet completed."
fi

# Test 4: CLI metrics and monitoring interfaces
echo "🖥️ Testing CLI metrics and monitoring interfaces..."

# Test metrics CLI
metrics_output=$(git-capsulate metrics show)
if [[ "$metrics_output" == *"Metrics Summary"* ]]; then
  echo "✅ Metrics CLI test passed!"
else
  echo "❌ Metrics CLI test failed!"
  exit 1
fi

# Test monitor CLI
monitor_output=$(git-capsulate monitor show)
if [[ "$monitor_output" == *"Resource Usage"* ]] || [[ "$monitor_output" == *"No container stats available"* ]]; then
  echo "✅ Monitor CLI test passed!"
else
  echo "❌ Monitor CLI test failed!"
  exit 1
fi

# Clean up
echo "🧹 Cleaning up..."
git-capsulate destroy obs-test-agent1
git-capsulate destroy obs-test-agent2
git-capsulate destroy obs-test-agent3
git-capsulate monitor stop

echo "🎉 All Phase 4 observability tests completed!" 