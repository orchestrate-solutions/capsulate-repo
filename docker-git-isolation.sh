#!/bin/bash
# Docker-based Git Isolation for AI Agents
# This script creates isolated environments for multiple AI agents to work on the same repo

# Configuration
REPO_URL="https://github.com/yourusername/yourrepo.git"
BASE_BRANCH="main"
HOST_PROJECT_DIR="$(pwd)/project-files"
SSH_KEY_PATH="$HOME/.ssh"
SHARE_AUTH=true

# Parse command line arguments
while [ "$1" != "" ]; do
  case $1 in
    --repo-url)      shift; REPO_URL=$1 ;;
    --branch)        shift; BASE_BRANCH=$1 ;;
    --project-dir)   shift; HOST_PROJECT_DIR=$1 ;;
    --ssh-path)      shift; SSH_KEY_PATH=$1 ;;
    --no-shared-auth) SHARE_AUTH=false ;;
    --help)          
      echo "Usage: $0 [options]"
      echo "  --repo-url URL     Repository URL"
      echo "  --branch BRANCH    Base branch"
      echo "  --project-dir DIR  Host directory for files"
      echo "  --ssh-path DIR     Path to SSH keys"
      echo "  --no-shared-auth   Disable shared authentication"
      exit 0
      ;;
  esac
  shift
done

# Create base image
echo "🔨 Building base Docker image for Git operations..."
docker build -t ai-agent-base -f- . <<EOF
FROM ubuntu:latest
RUN apt-get update && apt-get install -y git curl openssh-client
RUN mkdir -p /root/.ssh && chmod 700 /root/.ssh
RUN echo "StrictHostKeyChecking no" >> /etc/ssh/ssh_config
WORKDIR /workspace
EOF

# Create project directory if it doesn't exist
mkdir -p "$HOST_PROJECT_DIR"

# Function to create and initialize an agent container
create_agent() {
  local agent_id=$1
  local branch_name="feature/agent-$agent_id"
  local agent_dir="$HOST_PROJECT_DIR/agent-$agent_id"
  
  echo "🚀 Creating container for Agent $agent_id..."
  
  # Create agent directory
  mkdir -p "$agent_dir"
  
  # Remove container if it already exists
  docker rm -f agent$agent_id 2>/dev/null
  
  # Build docker run command
  RUN_CMD="docker run -d --name agent$agent_id"
  
  # Add volume mounts and environment variables
  RUN_CMD+=" -v $agent_dir:/workspace"
  
  if [ "$SHARE_AUTH" = true ]; then
    RUN_CMD+=" -v $SSH_KEY_PATH:/root/.ssh:ro"
    RUN_CMD+=" -e GIT_SSH_COMMAND='ssh -o StrictHostKeyChecking=no'"
  else
    RUN_CMD+=" -v agent${agent_id}-ssh:/root/.ssh"
  fi
  
  RUN_CMD+=" -e GIT_AUTHOR_NAME='Agent $agent_id'"
  RUN_CMD+=" -e GIT_AUTHOR_EMAIL='agent${agent_id}@example.com'"
  RUN_CMD+=" -e GIT_COMMITTER_NAME='Agent $agent_id'"
  RUN_CMD+=" -e GIT_COMMITTER_EMAIL='agent${agent_id}@example.com'"
  RUN_CMD+=" ai-agent-base sleep infinity"
  
  # Run the container
  eval "$RUN_CMD"
  
  echo "📦 Cloning repository for Agent $agent_id..."
  
  # Convert HTTPS URLs to SSH for shared auth
  CLONE_URL="$REPO_URL"
  if [ "$SHARE_AUTH" = true ] && [[ "$REPO_URL" == https://github.com/* ]]; then
    CLONE_URL="git@github.com:${REPO_URL#https://github.com/}"
  fi
  
  docker exec agent$agent_id git clone "$CLONE_URL" repo
  
  echo "🌿 Creating branch $branch_name for Agent $agent_id..."
  docker exec agent$agent_id bash -c "cd repo && git checkout -b $branch_name"
  
  # Create a status file in the host directory
  cat > "$agent_dir/.git-status.md" <<EOL
# Git Status for Agent $agent_id ($branch_name)
Last updated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

Branch is ready for use.
EOL
  
  echo "✅ Agent $agent_id environment ready"
  echo "   - Container: agent$agent_id"
  echo "   - Branch: $branch_name"
  echo "   - Files at: $agent_dir"
  echo ""
}

# Function to run a Git command for a specific agent
run_git_command() {
  local agent_id=$1
  local command=$2
  
  echo "🔄 Agent $agent_id executing: git $command"
  docker exec agent$agent_id bash -c "cd repo && git $command"
  
  # Update status file
  local agent_dir="$HOST_PROJECT_DIR/agent-$agent_id"
  local status_file="$agent_dir/.git-status.md"
  
  if [ -d "$agent_dir" ]; then
    local branch_output=$(docker exec agent$agent_id bash -c "cd repo && git branch -v")
    local status_output=$(docker exec agent$agent_id bash -c "cd repo && git status")
    
    cat > "$status_file" <<EOL
# Git Status for Agent $agent_id
Last updated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

## Current Branch
$branch_output

## Status
$status_output
EOL
    
    echo "📋 Status file updated at $status_file"
  fi
}

# Function to get a summary of all agents
list_agents() {
  echo "🔍 Listing all Git agents..."
  
  for container in $(docker ps --filter "name=agent[0-9]+" --format "{{.Names}}"); do
    agent_id=${container#agent}
    branch=$(docker exec $container bash -c "cd repo && git branch --show-current" 2>/dev/null || echo "unknown")
    
    echo "Agent $agent_id:"
    echo "  - Container: $container"
    echo "  - Branch: $branch"
    echo "  - Files at: $HOST_PROJECT_DIR/agent-$agent_id"
  done
}

# Function to clean up an agent
remove_agent() {
  local agent_id=$1
  
  echo "🧹 Removing Agent $agent_id..."
  docker rm -f agent$agent_id
  
  echo "✅ Agent $agent_id removed"
}

# Main script execution
case "$1" in
  create)
    if [ -z "$2" ]; then
      echo "Error: Missing agent ID"
      echo "Usage: $0 create AGENT_ID"
      exit 1
    fi
    create_agent "$2"
    ;;
    
  exec)
    if [ -z "$2" ] || [ -z "$3" ]; then
      echo "Error: Missing arguments"
      echo "Usage: $0 exec AGENT_ID GIT_COMMAND"
      exit 1
    fi
    run_git_command "$2" "$3"
    ;;
    
  list)
    list_agents
    ;;
    
  remove)
    if [ -z "$2" ]; then
      echo "Error: Missing agent ID"
      echo "Usage: $0 remove AGENT_ID"
      exit 1
    fi
    remove_agent "$2"
    ;;
    
  *)
    echo "🚀 Git Isolation Script"
    echo ""
    echo "Usage:"
    echo "  $0 create AGENT_ID        Create new agent environment"
    echo "  $0 exec AGENT_ID COMMAND  Execute git command for agent"
    echo "  $0 list                   List all agents"
    echo "  $0 remove AGENT_ID        Remove an agent"
    echo ""
    echo "Configuration:"
    echo "  Repository: $REPO_URL"
    echo "  Base branch: $BASE_BRANCH"
    echo "  Project directory: $HOST_PROJECT_DIR"
    echo "  SSH keys: $SSH_KEY_PATH"
    echo "  Shared auth: $SHARE_AUTH"
    echo ""
    echo "Examples:"
    echo "  $0 create feature-x        # Create agent for feature-x"
    echo "  $0 exec feature-x \"status\" # Check git status"
    echo "  $0 exec feature-x \"add .\"  # Stage all changes"
    echo "  $0 exec feature-x \"commit -m 'Add new files'\" # Commit changes"
    echo "  $0 exec feature-x \"push origin feature/agent-feature-x\" # Push changes"
    ;;
esac 