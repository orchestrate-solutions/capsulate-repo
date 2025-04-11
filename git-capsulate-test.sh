#!/bin/bash
# Temporary test implementation of git-capsulate for validating Phase 1

if [ $# -lt 1 ]; then
  echo "Usage: $0 <command> [args...]"
  exit 1
fi

COMMAND=$1
shift

case $COMMAND in
  create)
    if [ $# -lt 1 ]; then
      echo "Usage: $0 create <agent-id>"
      exit 1
    fi
    AGENT_ID=$1
    CONTAINER_NAME="capsulate-$AGENT_ID"
    
    # Check if container already exists
    if docker ps -a | grep -q $CONTAINER_NAME; then
      echo "Agent with ID '$AGENT_ID' already exists"
      exit 1
    fi
    
    # Get SSH directory
    SSH_DIR="$HOME/.ssh"
    
    # Get current directory as workspace
    WORKSPACE_DIR="$(pwd)"
    
    # Create container
    echo "Creating agent '$AGENT_ID'..."
    docker run -d --name $CONTAINER_NAME \
      -v "$SSH_DIR:/root/.ssh:ro" \
      -v "$WORKSPACE_DIR:/workspace" \
      ubuntu:latest tail -f /dev/null
    
    # Install Git in the container
    docker exec $CONTAINER_NAME apt-get update
    docker exec $CONTAINER_NAME apt-get install -y git
    
    echo "Agent '$AGENT_ID' created successfully"
    ;;
    
  exec)
    if [ $# -lt 2 ]; then
      echo "Usage: $0 exec <agent-id> <command>"
      exit 1
    fi
    AGENT_ID=$1
    shift
    COMMAND="$@"
    CONTAINER_NAME="capsulate-$AGENT_ID"
    
    # Check if container exists
    if ! docker ps | grep -q $CONTAINER_NAME; then
      echo "Agent '$AGENT_ID' not found or not running"
      exit 1
    fi
    
    # Execute command
    docker exec $CONTAINER_NAME sh -c "$COMMAND"
    ;;
    
  destroy)
    if [ $# -lt 1 ]; then
      echo "Usage: $0 destroy <agent-id>"
      exit 1
    fi
    AGENT_ID=$1
    CONTAINER_NAME="capsulate-$AGENT_ID"
    
    # Check if container exists
    if ! docker ps -a | grep -q $CONTAINER_NAME; then
      echo "Agent '$AGENT_ID' not found"
      exit 1
    fi
    
    # Stop and remove container
    echo "Destroying agent '$AGENT_ID'..."
    docker stop $CONTAINER_NAME
    docker rm $CONTAINER_NAME
    
    echo "Agent '$AGENT_ID' destroyed successfully"
    ;;
    
  *)
    echo "Unknown command: $COMMAND"
    echo "Available commands: create, exec, destroy"
    exit 1
    ;;
esac 