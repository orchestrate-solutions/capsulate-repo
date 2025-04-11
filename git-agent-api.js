/**
 * Git Agent API Server
 * 
 * A simple Express server that provides an HTTP API for managing Git-isolated agents
 */

const express = require('express');
const bodyParser = require('body-parser');
const GitAgentManager = require('./git-agent-manager');

const app = express();
const PORT = process.env.PORT || 3000;

// Parse JSON request bodies
app.use(bodyParser.json());

// Create Git Agent Manager
const manager = new GitAgentManager({
  repoUrl: process.env.REPO_URL || 'https://github.com/user/repo.git',
  baseBranch: process.env.BASE_BRANCH || 'main'
});

// Initialize on startup
let initialized = false;
async function ensureInitialized() {
  if (!initialized) {
    await manager.initialize();
    initialized = true;
  }
}

// Routes

// Get status
app.get('/status', async (req, res) => {
  try {
    await ensureInitialized();
    res.json({
      status: 'online',
      agents: Object.fromEntries(manager.agents),
      repository: manager.repoUrl,
      baseBranch: manager.baseBranch
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Create new agent
app.post('/agents', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId, name, email, branchPrefix } = req.body;
    
    if (!agentId) {
      return res.status(400).json({ error: 'agentId is required' });
    }
    
    const agent = await manager.createAgent(agentId, { 
      name, 
      email, 
      branchPrefix 
    });
    
    res.status(201).json(agent);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// List all agents
app.get('/agents', async (req, res) => {
  try {
    await ensureInitialized();
    
    const agents = Array.from(manager.agents.values());
    res.json(agents);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get agent by ID
app.get('/agents/:agentId', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId } = req.params;
    
    if (!manager.agents.has(agentId)) {
      return res.status(404).json({ error: `Agent ${agentId} not found` });
    }
    
    res.json(manager.agents.get(agentId));
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Execute Git command
app.post('/agents/:agentId/git', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId } = req.params;
    const { command } = req.body;
    
    if (!command) {
      return res.status(400).json({ error: 'command is required' });
    }
    
    if (!manager.agents.has(agentId)) {
      return res.status(404).json({ error: `Agent ${agentId} not found` });
    }
    
    const result = await manager.executeGitCommand(agentId, command);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Execute file operation
app.post('/agents/:agentId/file', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId } = req.params;
    const { operation, path, content } = req.body;
    
    if (!operation || !path) {
      return res.status(400).json({ 
        error: 'operation and path are required' 
      });
    }
    
    if (operation === 'write' && content === undefined) {
      return res.status(400).json({ 
        error: 'content is required for write operation' 
      });
    }
    
    if (!manager.agents.has(agentId)) {
      return res.status(404).json({ error: `Agent ${agentId} not found` });
    }
    
    const result = await manager.executeFileOperation(
      agentId, 
      operation, 
      { path, content }
    );
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Push changes
app.post('/agents/:agentId/push', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId } = req.params;
    const { remote, credentialsFile } = req.body;
    
    if (!manager.agents.has(agentId)) {
      return res.status(404).json({ error: `Agent ${agentId} not found` });
    }
    
    const result = await manager.pushChanges(agentId, { 
      remote, 
      credentialsFile 
    });
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Remove agent
app.delete('/agents/:agentId', async (req, res) => {
  try {
    await ensureInitialized();
    
    const { agentId } = req.params;
    
    if (!manager.agents.has(agentId)) {
      return res.status(404).json({ error: `Agent ${agentId} not found` });
    }
    
    await manager.removeAgent(agentId);
    res.json({ success: true, message: `Agent ${agentId} removed` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Cleanup all agents
app.post('/cleanup', async (req, res) => {
  try {
    await ensureInitialized();
    
    await manager.cleanup();
    res.json({ success: true, message: 'All agents removed' });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Start server
app.listen(PORT, async () => {
  try {
    await manager.initialize();
    initialized = true;
    console.log(`🚀 Git Agent API server running on port ${PORT}`);
  } catch (error) {
    console.error('Failed to initialize server:', error);
    process.exit(1);
  }
});

/**
 * Example API Usage:
 * 
 * Create Agent:
 * POST /agents
 * { "agentId": "1", "name": "AI Assistant 1" }
 * 
 * Execute Git Command:
 * POST /agents/1/git
 * { "command": "status" }
 * 
 * Write File:
 * POST /agents/1/file
 * { "operation": "write", "path": "feature.txt", "content": "New feature" }
 * 
 * Push Changes:
 * POST /agents/1/push
 * { "remote": "origin", "credentialsFile": "/path/to/credentials" }
 */ 