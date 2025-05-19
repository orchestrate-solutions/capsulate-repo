
# Local Branch Isolation Strategies Overview

This document outlines approaches to managing multiple isolated Git branches or workspaces **on a single local machine**. It covers solutions from simple Git-native tools to containerized environments inspired by CapsulateRepo.

---

## Goals

- Maintain multiple isolated branches/environments on one computer  
- Work on branches concurrently without changes leaking between them  
- Balance disk usage, complexity, and isolation needs  
- Optionally isolate dependencies alongside code

---

## Approaches

### 1. Git Worktree (Recommended for most cases)

- Uses `git worktree` to create multiple working directories linked to the same repository data  
- Each directory is a checked-out branch, allowing concurrent work  
- Efficient disk usage by sharing `.git` metadata internally  
- Simple to set up and use

**Example:**

```bash
cd /path/to/repo
git worktree add ../repo-feature feature-branch
```

---

### 2. Separate Clones

- Clone the repo into multiple folders, each at a different branch  
- Complete isolation of files and `.git` data  
- Higher disk space usage  
- Simple but less efficient

---

### 3. Docker Container Isolation (CapsulateRepo Style)

- Use containers with OverlayFS to create fully isolated environments  
- Containers share a read-only base repo layer, with changes stored in a diff layer  
- Dependency management across core, team, and container levels  
- High isolation and environment control  
- Best for parallel AI/human workflows or complex dependency needs  
- Requires Docker and more setup

---

## Summary Table

| Approach                 | Isolation Level          | Disk Usage   | Complexity      | Recommended For                        |
|--------------------------|-------------------------|--------------|-----------------|-------------------------------------|
| **Git Worktree**          | Good (separate dirs)    | Low          | Low             | Concurrent branch work locally      |
| **Separate Clones**       | Full (separate repos)   | High         | Very low        | Full isolation, different setups    |
| **Docker (CapsulateRepo)**| Full OS/container isolation | Medium-high | High            | Advanced parallel dev and experiments|

---

## Next Steps

- For straightforward local branching with isolation, try `git worktree`.  
- For full OS-level isolation or experimental dependency management, consider Docker-based CapsulateRepo environments.  
- Decide based on disk space, number of branches, and complexity tolerance.

---

If you want, I can help you with concrete commands or scripts for any of these options.
