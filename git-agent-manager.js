/**
 * Git Agent Manager
 * 
 * A Node.js module for managing Git-isolated AI agents using Docker
 */

const { exec } = require('child_process');
const util = require('util');
const path = require('path');
const fs = require('fs');
const os = require('os');
const execPromise = util.promisify(exec);

class GitAgentManager {
  /**
   * Create a new Git Agent Manager
   * @param {Object} options - Configuration options
   * @param {string} options.repoUrl - GitHub repository URL
   * @param {string} options.baseBranch - Base branch to work from
   * @param {string} options.baseImage - Docker image to use (default: builds ubuntu with git)
   * @param {string} options.hostProjectDir - Host directory to mount in containers (for two-way file sharing)
   * @param {string} options.sshKeyPath - Path to SSH keys (default: ~/.ssh)
   * @param {boolean} options.shareAuth - Whether to share host authentication (default: true)
   */
  constructor(options) {
    this.repoUrl = options.repoUrl;
    this.baseBranch = options.baseBranch || 'main';
    this.baseImage = options.baseImage || 'ai-agent-base';
    this.hostProjectDir = options.hostProjectDir || '';
    this.sshKeyPath = options.sshKeyPath || path.join(os.homedir(), '.ssh');
    this.shareAuth = options.shareAuth !== false; // Default to true
    this.agents = new Map();
    this.initialized = false;
  }

  /**
   * Initialize the system
   */
  async initialize() {
    if (this.initialized) return;
    
    try {
      // Build base Docker image if not provided
      if (this.baseImage === 'ai-agent-base') {
        console.log('🔨 Building base Docker image...');
        const dockerfile = `
          FROM ubuntu:latest
          RUN apt-get update && apt-get install -y git curl openssh-client
          RUN mkdir -p /root/.ssh && chmod 700 /root/.ssh
          RUN echo "StrictHostKeyChecking no" >> /etc/ssh/ssh_config
          WORKDIR /workspace
        `;
        
        await execPromise(`
          docker build -t ${this.baseImage} -f- . <<EOF
          ${dockerfile}
          EOF
        `);
      }
      
      // Create host project directory if specified and doesn't exist
      if (this.hostProjectDir && !fs.existsSync(this.hostProjectDir)) {
        fs.mkdirSync(this.hostProjectDir, { recursive: true });
      }
      
      this.initialized = true;
      console.log('✅ Git Agent Manager initialized');
    } catch (error) {
      console.error('❌ Failed to initialize:', error);
      throw error;
    }
  }

  /**
   * Create a new agent with isolated Git environment
   * @param {string} agentId - Unique identifier for the agent
   * @param {Object} options - Agent options
   * @param {string} options.name - Agent name
   * @param {string} options.email - Agent email
   * @param {string} options.branchPrefix - Branch name prefix (default: 'feature/agent-')
   * @param {string} options.hostDir - Host directory for this agent (overrides hostProjectDir)
   * @returns {Promise<Object>} - Agent information
   */
  async createAgent(agentId, options = {}) {
    if (!this.initialized) await this.initialize();
    
    const name = options.name || `Agent ${agentId}`;
    const email = options.email || `agent-${agentId}@example.com`;
    const branchPrefix = options.branchPrefix || 'feature/agent-';
    const branchName = `${branchPrefix}${agentId}`;
    const containerName = `agent-${agentId}`;
    
    // Determine host directory for this agent
    let hostDir = options.hostDir;
    if (!hostDir && this.hostProjectDir) {
      hostDir = path.join(this.hostProjectDir, `agent-${agentId}`);
      if (!fs.existsSync(hostDir)) {
        fs.mkdirSync(hostDir, { recursive: true });
      }
    }
    
    try {
      // Remove container if it already exists
      await execPromise(`docker rm -f ${containerName} 2>/dev/null || true`);
      
      // Create container with volume mounts
      console.log(`🚀 Creating container for ${name}...`);
      let runCommand = `
        docker run -d --name ${containerName} \\
          -e GIT_AUTHOR_NAME="${name}" \\
          -e GIT_AUTHOR_EMAIL="${email}" \\
          -e GIT_COMMITTER_NAME="${name}" \\
          -e GIT_COMMITTER_EMAIL="${email}"
      `;
      
      // Add SSH key mount for shared auth if enabled
      if (this.shareAuth) {
        runCommand += ` \\\n          -v ${this.sshKeyPath}:/root/.ssh:ro`;
        runCommand += ` \\\n          -e GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no"`;
      } else {
        // If not sharing auth, use a dedicated volume for this agent
        runCommand += ` \\\n          -v agent-${agentId}-ssh:/root/.ssh`;
      }
      
      // Mount host directory if specified (two-way file sharing)
      if (hostDir) {
        runCommand += ` \\\n          -v ${hostDir}:/workspace`;
      } else {
        // Otherwise use a docker volume (one-way, container-only)
        runCommand += ` \\\n          -v agent-${agentId}-workspace:/workspace`;
      }
      
      // Finish the command with the image and a long-running process
      runCommand += ` \\\n          ${this.baseImage} sleep infinity`;
      
      await execPromise(runCommand);
      
      // Clone repository
      console.log(`📦 Cloning repository for ${name}...`);
      
      // Choose between SSH and HTTPS for Git clone based on repo URL
      let cloneUrl = this.repoUrl;
      if (this.shareAuth && cloneUrl.startsWith('https://github.com/')) {
        // Convert HTTPS URLs to SSH if we're sharing auth
        cloneUrl = `git@github.com:${cloneUrl.replace('https://github.com/', '')}`;
      }
      
      await execPromise(`docker exec ${containerName} git clone ${cloneUrl} repo`);
      
      // Create branch
      console.log(`🌿 Creating branch ${branchName}...`);
      await execPromise(`docker exec ${containerName} bash -c "cd repo && git checkout -b ${branchName}"`);
      
      // Store agent info
      const agent = {
        id: agentId,
        name,
        email,
        container: containerName,
        branch: branchName,
        hostDir: hostDir || null
      };
      
      this.agents.set(agentId, agent);
      console.log(`✅ Agent ${agentId} environment ready`);
      console.log(`   - Container: ${containerName}`);
      console.log(`   - Branch: ${branchName}`);
      if (hostDir) {
        console.log(`   - Files at: ${hostDir}`);
      }
      
      return agent;
    } catch (error) {
      console.error(`❌ Failed to create agent ${agentId}:`, error);
      throw error;
    }
  }

  /**
   * Execute a Git command for a specific agent
   * @param {string} agentId - Agent identifier
   * @param {string} command - Git command to execute
   * @returns {Promise<Object>} - Command result with stdout and stderr
   */
  async executeGitCommand(agentId, command) {
    if (!this.agents.has(agentId)) {
      throw new Error(`Agent ${agentId} does not exist`);
    }
    
    const agent = this.agents.get(agentId);
    console.log(`🔄 ${agent.name} executing: git ${command}`);
    
    try {
      const { stdout, stderr } = await execPromise(
        `docker exec ${agent.container} bash -c "cd repo && git ${command}"`
      );
      
      return { stdout, stderr };
    } catch (error) {
      console.error(`❌ Git command failed for agent ${agentId}:`, error);
      throw error;
    }
  }

  /**
   * Execute a file operation for a specific agent
   * @param {string} agentId - Agent identifier
   * @param {string} operation - File operation (e.g., 'write', 'read', 'delete')
   * @param {Object} options - Operation options
   * @returns {Promise<Object>} - Operation result
   */
  async executeFileOperation(agentId, operation, options) {
    if (!this.agents.has(agentId)) {
      throw new Error(`Agent ${agentId} does not exist`);
    }
    
    const agent = this.agents.get(agentId);
    
    try {
      switch (operation) {
        case 'write': {
          const { path, content } = options;
          console.log(`📝 ${agent.name} writing to ${path}...`);
          
          // Escape content for shell
          const escapedContent = content.replace(/'/g, "'\\''");
          
          await execPromise(
            `docker exec ${agent.container} bash -c "cd repo && echo '${escapedContent}' > ${path}"`
          );
          return { success: true, path };
        }
        
        case 'read': {
          const { path } = options;
          console.log(`📖 ${agent.name} reading ${path}...`);
          
          const { stdout } = await execPromise(
            `docker exec ${agent.container} bash -c "cd repo && cat ${path}"`
          );
          return { content: stdout, path };
        }
        
        case 'delete': {
          const { path } = options;
          console.log(`🗑️ ${agent.name} deleting ${path}...`);
          
          await execPromise(
            `docker exec ${agent.container} bash -c "cd repo && rm ${path}"`
          );
          return { success: true, path };
        }
        
        default:
          throw new Error(`Unknown file operation: ${operation}`);
      }
    } catch (error) {
      console.error(`❌ File operation failed for agent ${agentId}:`, error);
      throw error;
    }
  }

  /**
   * Push changes from an agent to GitHub
   * @param {string} agentId - Agent identifier
   * @param {Object} options - Push options
   * @param {string} options.remote - Remote name (default: 'origin')
   * @returns {Promise<Object>} - Push result
   */
  async pushChanges(agentId, options = {}) {
    if (!this.agents.has(agentId)) {
      throw new Error(`Agent ${agentId} does not exist`);
    }
    
    const agent = this.agents.get(agentId);
    const remote = options.remote || 'origin';
    
    console.log(`📤 ${agent.name} pushing to ${remote}/${agent.branch}...`);
    
    try {
      // Push changes
      const { stdout, stderr } = await execPromise(
        `docker exec ${agent.container} bash -c "cd repo && git push ${remote} ${agent.branch}"`
      );
      
      return { stdout, stderr, branch: agent.branch };
    } catch (error) {
      console.error(`❌ Push failed for agent ${agentId}:`, error);
      throw error;
    }
  }

  /**
   * Update or create agent status file in host directory
   * @param {string} agentId - Agent identifier
   * @returns {Promise<void>}
   */
  async updateStatusFile(agentId) {
    if (!this.agents.has(agentId)) {
      throw new Error(`Agent ${agentId} does not exist`);
    }
    
    const agent = this.agents.get(agentId);
    
    // Skip if no host directory
    if (!agent.hostDir) return;
    
    try {
      const status = await this.executeGitCommand(agentId, 'status');
      const branch = await this.executeGitCommand(agentId, 'branch -v');
      
      const statusText = `# Git Status for ${agent.name} (${agent.branch})
Last updated: ${new Date().toISOString()}

## Current Branch
${branch.stdout}

## Status
${status.stdout}
`;
      
      const statusPath = path.join(agent.hostDir, '.git-status.md');
      fs.writeFileSync(statusPath, statusText);
      
      console.log(`📋 Status file updated at ${statusPath}`);
    } catch (error) {
      console.warn(`⚠️ Failed to update status file: ${error.message}`);
    }
  }

  /**
   * Clean up and remove an agent
   * @param {string} agentId - Agent identifier
   * @returns {Promise<boolean>} - Success status
   */
  async removeAgent(agentId) {
    if (!this.agents.has(agentId)) {
      throw new Error(`Agent ${agentId} does not exist`);
    }
    
    const agent = this.agents.get(agentId);
    console.log(`🧹 Removing agent ${agentId}...`);
    
    try {
      await execPromise(`docker rm -f ${agent.container}`);
      this.agents.delete(agentId);
      return true;
    } catch (error) {
      console.error(`❌ Failed to remove agent ${agentId}:`, error);
      throw error;
    }
  }

  /**
   * Clean up all agents and resources
   * @returns {Promise<boolean>} - Success status
   */
  async cleanup() {
    console.log('🧹 Cleaning up all agents...');
    
    try {
      for (const agentId of this.agents.keys()) {
        await this.removeAgent(agentId);
      }
      
      // Optional: remove the base image
      // await execPromise(`docker rmi ${this.baseImage}`);
      
      return true;
    } catch (error) {
      console.error('❌ Cleanup failed:', error);
      throw error;
    }
  }
}

module.exports = GitAgentManager;

// Example usage
async function example() {
  const manager = new GitAgentManager({
    repoUrl: 'https://github.com/user/repo.git',
    baseBranch: 'main',
    // Share host authentication
    shareAuth: true,
    // Create a directory for two-way file sharing
    hostProjectDir: './project-files'
  });
  
  await manager.initialize();
  
  // Create two agents with shared files
  const agent1 = await manager.createAgent('1', { name: 'AIAgent1' });
  const agent2 = await manager.createAgent('2', { name: 'AIAgent2' });
  
  // Agent 1 makes changes
  await manager.executeFileOperation('1', 'write', {
    path: 'agent1-feature.txt',
    content: 'New feature implemented by Agent 1'
  });
  
  await manager.executeGitCommand('1', 'add .');
  await manager.executeGitCommand('1', 'commit -m "Add new feature"');
  
  // Agent 2 makes different changes
  await manager.executeFileOperation('2', 'write', {
    path: 'agent2-bugfix.txt',
    content: 'Bug fix implemented by Agent 2'
  });
  
  await manager.executeGitCommand('2', 'add .');
  await manager.executeGitCommand('2', 'commit -m "Fix critical bug"');
  
  // Push changes (using host's SSH keys)
  await manager.pushChanges('1');
  await manager.pushChanges('2');
  
  // Clean up when done
  // await manager.cleanup();
}

// Uncomment to run the example
// example().catch(console.error); 