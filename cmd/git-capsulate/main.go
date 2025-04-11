package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/capsulate-repo/pkg/agent"
	"github.com/your-org/capsulate-repo/pkg/metrics"
	"github.com/your-org/capsulate-repo/pkg/monitor"
	"github.com/your-org/capsulate-repo/pkg/tracing"
)

func main() {
	// Initialize the root command
	rootCmd := &cobra.Command{
		Use:   "git-capsulate",
		Short: "Git isolation using Docker containers",
		Long:  `Git-capsulate provides isolated Git environments using Docker containers for parallel development.`,
	}

	// Add create command
	createCmd := &cobra.Command{
		Use:   "create [agent-id]",
		Short: "Create a new Git isolation container",
		Long:  `Create a new container with Git isolation for development.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Get command-line flags
			repoURL, _ := cmd.Flags().GetString("repo")
			branch, _ := cmd.Flags().GetString("branch")
			depth, _ := cmd.Flags().GetInt("depth")
			depLevel, _ := cmd.Flags().GetString("dependency-level")
			teamID, _ := cmd.Flags().GetString("team-id")
			overrideDepsStr, _ := cmd.Flags().GetString("override-deps")
			useOverlay, _ := cmd.Flags().GetBool("use-overlay")
			
			// Parse override dependencies
			var overrideDeps []string
			if overrideDepsStr != "" {
				overrideDeps = strings.Split(overrideDepsStr, ",")
			}
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Create agent configuration
			config := agent.AgentConfig{
				ID:              agentID,
				DependencyLevel: depLevel,
				TeamID:          teamID,
				OverrideDeps:    overrideDeps,
				UseOverlay:      useOverlay,
				RepoURL:         repoURL,
				Branch:          branch,
				Depth:           depth,
			}

			// Create the agent
			if err := manager.Create(config); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Agent '%s' created successfully\n", agentID)
		},
	}

	// Add create command flags
	createCmd.Flags().StringP("repo", "r", "", "Git repository URL to clone")
	createCmd.Flags().StringP("branch", "b", "", "Branch to checkout")
	createCmd.Flags().IntP("depth", "d", 0, "Depth for shallow clones (0 for full clone)")
	createCmd.Flags().String("dependency-level", "container", "Dependency isolation level (core, team, container)")
	createCmd.Flags().String("team-id", "", "Team identifier for team-level dependencies")
	createCmd.Flags().String("override-deps", "", "Comma-separated list of dependencies to override")
	createCmd.Flags().Bool("use-overlay", false, "Use overlay filesystem for efficient storage")

	// Add destroy command
	destroyCmd := &cobra.Command{
		Use:   "destroy [agent-id]",
		Short: "Destroy a Git isolation container",
		Long:  `Stop and remove a Git isolation container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Destroy the agent
			if err := manager.Destroy(agentID); err != nil {
				fmt.Fprintf(os.Stderr, "Error destroying agent: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Agent '%s' destroyed successfully\n", agentID)
		},
	}

	// Add exec command
	execCmd := &cobra.Command{
		Use:   "exec [agent-id] [command]",
		Short: "Execute a command in a Git isolation container",
		Long:  `Run a command inside a Git isolation container.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			command := args[1]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Execute the command
			output, err := manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
				os.Exit(1)
			}

			fmt.Print(output)
		},
	}

	// Add Git branch command
	branchCmd := &cobra.Command{
		Use:   "branch [agent-id] [branch-name]",
		Short: "Create a Git branch in a container",
		Long:  `Create a new Git branch in a container and optionally check it out.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			branchName := args[1]
			
			checkout, _ := cmd.Flags().GetBool("checkout")
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Create the branch
			if err := manager.CreateBranch(agentID, branchName, checkout); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating branch: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Branch '%s' created", branchName)
			if checkout {
				fmt.Print(" and checked out")
			}
			fmt.Println()
		},
	}

	// Add branch command flags
	branchCmd.Flags().BoolP("checkout", "c", false, "Checkout the new branch after creation")

	// Add Git checkout command
	checkoutCmd := &cobra.Command{
		Use:   "checkout [agent-id] [branch-name]",
		Short: "Checkout a Git branch in a container",
		Long:  `Switch to a different Git branch in a container.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			branchName := args[1]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Checkout the branch
			if err := manager.CheckoutBranch(agentID, branchName); err != nil {
				fmt.Fprintf(os.Stderr, "Error checking out branch: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Switched to branch '%s'\n", branchName)
		},
	}

	// Add Git status command
	statusCmd := &cobra.Command{
		Use:   "status [agent-id]",
		Short: "Show Git status in a container",
		Long:  `Display Git status information for a container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Get Git status
			status, err := manager.GetGitStatus(agentID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting Git status: %v\n", err)
				os.Exit(1)
			}

			// Print status
			fmt.Printf("Branch: %s\n", status.Branch)
			fmt.Printf("Commit: %s\n", status.CurrentCommit)
			fmt.Printf("Ahead: %d, Behind: %d\n\n", status.AheadCount, status.BehindCount)
			
			fmt.Println("Modified files:")
			for _, file := range status.ModifiedFiles {
				fmt.Printf("  - %s\n", file)
			}
			
			fmt.Println("\nUntracked files:")
			for _, file := range status.UntrackedFiles {
				fmt.Printf("  - %s\n", file)
			}
		},
	}

	// Add dependency commands
	
	// List dependencies command
	listDepsCmd := &cobra.Command{
		Use:   "list-deps [agent-id]",
		Short: "List dependencies in a container",
		Long:  `Display the dependencies available in a container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Get command to list all dependencies
			command := "ls -la /workspace/node_modules/"
			output, err := manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing dependencies: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Dependencies for agent '%s':\n", agentID)
			fmt.Println(output)
		},
	}

	// Add dependency command
	addDepCmd := &cobra.Command{
		Use:   "add-dep [agent-id] [package]",
		Short: "Add a dependency to a container",
		Long:  `Add a new dependency to a container's isolated environment.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			packageName := args[1]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Create a stub directory for the package in the container-deps
			command := fmt.Sprintf("mkdir -p /workspace/container-deps/%s", packageName)
			_, err = manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding dependency: %v\n", err)
				os.Exit(1)
			}

			// Create a version file in the package directory
			command = fmt.Sprintf("echo '1.0.0' > /workspace/container-deps/%s/version", packageName)
			_, err = manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting dependency version: %v\n", err)
				os.Exit(1)
			}

			// Create symbolic link in node_modules
			command = fmt.Sprintf("ln -sf /workspace/container-deps/%s /workspace/node_modules/%s", packageName, packageName)
			_, err = manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error linking dependency: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Added dependency '%s' to agent '%s'\n", packageName, agentID)
		},
	}

	// Add overlay filesystem commands
	
	// Overlay status command
	overlayStatusCmd := &cobra.Command{
		Use:   "overlay-status [agent-id]",
		Short: "Show overlay filesystem status",
		Long:  `Display information about the overlay filesystem for a container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Get SSH directory for auth
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
				os.Exit(1)
			}
			sshDir := filepath.Join(homeDir, ".ssh")
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create agent manager
			manager, err := agent.NewManager(sshDir, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent manager: %v\n", err)
				os.Exit(1)
			}

			// Check if the agent uses overlay
			command := "if mount | grep -q 'overlay on /workspace/merged'; then echo 'enabled'; else echo 'disabled'; fi"
			output, err := manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking overlay status: %v\n", err)
				os.Exit(1)
			}

			isEnabled := strings.TrimSpace(output) == "enabled"
			
			fmt.Printf("Overlay filesystem status for agent '%s':\n", agentID)
			if isEnabled {
				fmt.Println("Status: Enabled")
				
				// Get base layer file count
				baseCmd := "find /workspace/base -type f | wc -l"
				baseCount, err := manager.Exec(agentID, baseCmd)
				if err == nil {
					fmt.Printf("Base layer files: %s", baseCount)
				}
				
				// Get diff layer file count
				diffCmd := "find /workspace/diff -type f | wc -l"
				diffCount, err := manager.Exec(agentID, diffCmd)
				if err == nil {
					fmt.Printf("Diff layer files: %s", diffCount)
				}
				
				// Get total file count
				mergedCmd := "find /workspace/merged -type f | wc -l"
				mergedCount, err := manager.Exec(agentID, mergedCmd)
				if err == nil {
					fmt.Printf("Total files: %s", mergedCount)
				}
			} else {
				fmt.Println("Status: Disabled")
				fmt.Println("This agent is not using an overlay filesystem.")
			}
		},
	}

	// Add team commands

	// Create team command
	createTeamCmd := &cobra.Command{
		Use:   "create-team [team-id]",
		Short: "Create a new team",
		Long:  `Create a new team for sharing dependencies.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			teamID := args[0]
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create team directory
			teamPath := filepath.Join(workspaceDir, ".capsulate", "dependencies", "team", teamID)
			if err := os.MkdirAll(teamPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating team directory: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Team '%s' created successfully\n", teamID)
		},
	}
	
	// Add team dependency command
	addTeamDepCmd := &cobra.Command{
		Use:   "add-team-dep [team-id] [package]",
		Short: "Add a team dependency",
		Long:  `Add a dependency to a team's shared dependencies.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			teamID := args[0]
			packageName := args[1]
			
			// Get current working directory as workspace
			workspaceDir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create package directory in team dependencies
			packagePath := filepath.Join(workspaceDir, ".capsulate", "dependencies", "team", teamID, packageName)
			if err := os.MkdirAll(packagePath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating package directory: %v\n", err)
				os.Exit(1)
			}
			
			// Create a version file
			versionFile := filepath.Join(packagePath, "version")
			if err := os.WriteFile(versionFile, []byte("1.0.0"), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating version file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Added dependency '%s' to team '%s'\n", packageName, teamID)
		},
	}

	// Add metrics command
	metricsCmd := &cobra.Command{
		Use:   "metrics [subcommand]",
		Short: "Manage metrics and monitoring",
		Long:  `Commands for metrics collection, monitoring, and observability.`,
	}
	
	// Add metrics commands
	metricsShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show collected metrics",
		Long:  `Display a summary of collected metrics.`,
		Run: func(cmd *cobra.Command, args []string) {
			format, _ := cmd.Flags().GetString("format")
			
			if format == "json" {
				jsonSummary, err := metrics.GetSummaryJSON()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating metrics summary: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(jsonSummary)
			} else {
				summary := metrics.GetSummary()
				fmt.Println("📊 Metrics Summary:")
				fmt.Println("=====================================")
				
				for category, catSummary := range summary {
					fmt.Printf("🔹 Category: %s\n", category)
					fmt.Printf("  Total operations: %d\n", catSummary.TotalCount)
					if catSummary.AvgDuration > 0 {
						fmt.Printf("  Average duration: %.2f ms\n", catSummary.AvgDuration)
						fmt.Printf("  Min/Max duration: %.2f ms / %.2f ms\n", catSummary.MinDuration, catSummary.MaxDuration)
					}
					fmt.Println("  Operations:")
					for opName, opStats := range catSummary.Operations {
						fmt.Printf("    - %s: %d operations", opName, opStats.Count)
						if opStats.AvgDuration > 0 {
							fmt.Printf(", avg: %.2f ms", opStats.AvgDuration)
						}
						fmt.Println()
					}
					fmt.Println()
				}
			}
		},
	}
	metricsShowCmd.Flags().String("format", "text", "Output format (text or json)")
	
	metricsClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear collected metrics",
		Long:  `Clear all collected metrics from memory.`,
		Run: func(cmd *cobra.Command, args []string) {
			metrics.Clear()
			fmt.Println("✅ Metrics cleared")
		},
	}
	
	// Add monitoring commands
	monitorCmd := &cobra.Command{
		Use:   "monitor [subcommand]",
		Short: "Monitor agent containers",
		Long:  `Commands for monitoring agent containers.`,
	}
	
	monitorShowCmd := &cobra.Command{
		Use:   "show [agent-id]",
		Short: "Show resource usage stats",
		Long:  `Display resource usage statistics for agent containers.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			format, _ := cmd.Flags().GetString("format")
			
			var stats interface{}
			if len(args) > 0 {
				// Show stats for a specific agent
				agentID := args[0]
				stats = monitor.GetContainerStatsByAgentID(agentID)
				if stats == nil || len(stats.([]*monitor.ContainerStats)) == 0 {
					fmt.Printf("No stats available for agent '%s'\n", agentID)
					os.Exit(0)
				}
			} else {
				// Show stats for all agents
				stats = monitor.GetAllContainerStats()
				if stats == nil || len(stats.(map[string]*monitor.ContainerStats)) == 0 {
					fmt.Println("No container stats available")
					os.Exit(0)
				}
			}
			
			if format == "json" {
				jsonData, err := json.MarshalIndent(stats, "", "  ")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error marshaling stats to JSON: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(string(jsonData))
			} else {
				if len(args) > 0 {
					// Display stats for a specific agent
					agentStats := stats.([]*monitor.ContainerStats)
					fmt.Printf("📊 Resource Usage for Agent '%s':\n", args[0])
					fmt.Println("==========================================")
					for _, stat := range agentStats {
						displayContainerStats(stat)
					}
				} else {
					// Display stats for all agents
					allStats := stats.(map[string]*monitor.ContainerStats)
					fmt.Println("📊 Resource Usage for All Agents:")
					fmt.Println("==========================================")
					for _, stat := range allStats {
						fmt.Printf("🔹 Agent: %s\n", stat.AgentID)
						displayContainerStats(stat)
						fmt.Println()
					}
				}
			}
		},
	}
	monitorShowCmd.Flags().String("format", "text", "Output format (text or json)")
	
	monitorStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start container monitoring",
		Long:  `Start collecting resource usage statistics for agent containers.`,
		Run: func(cmd *cobra.Command, args []string) {
			monitor.Start()
			fmt.Println("✅ Monitoring started")
		},
	}
	
	monitorStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop container monitoring",
		Long:  `Stop collecting resource usage statistics for agent containers.`,
		Run: func(cmd *cobra.Command, args []string) {
			monitor.Stop()
			fmt.Println("✅ Monitoring stopped")
		},
	}
	
	// Add tracing commands
	tracesCmd := &cobra.Command{
		Use:   "traces",
		Short: "Manage traces and spans",
		Long:  `Commands for managing distributed traces and spans.`,
		Run: func(cmd *cobra.Command, args []string) {
			activeSpans := tracing.GetActiveSpans()
			
			if len(activeSpans) == 0 {
				fmt.Println("No active traces")
				return
			}
			
			fmt.Printf("🔍 Active Traces: %d\n", len(activeSpans))
			fmt.Println("==========================================")
			
			// Group spans by trace ID
			traceMap := make(map[string][]*tracing.Span)
			for _, span := range activeSpans {
				traceID := span.Context.TraceID
				traceMap[traceID] = append(traceMap[traceID], span)
			}
			
			for traceID, spans := range traceMap {
				fmt.Printf("Trace ID: %s\n", traceID)
				for _, span := range spans {
					fmt.Printf("  - Span: %s (ID: %s)\n", span.Name, span.Context.SpanID)
					fmt.Printf("    Started: %s\n", span.StartTime.Format(time.RFC3339))
					fmt.Printf("    Status: %d\n", span.Status.Code)
					if len(span.Attributes) > 0 {
						fmt.Println("    Attributes:")
						for k, v := range span.Attributes {
							fmt.Printf("      %s: %v\n", k, v)
						}
					}
					fmt.Println()
				}
			}
		},
	}
	
	// Register commands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(statusCmd)
	
	// Register dependency commands
	rootCmd.AddCommand(listDepsCmd)
	rootCmd.AddCommand(addDepCmd)
	
	// Register overlay commands
	rootCmd.AddCommand(overlayStatusCmd)
	
	// Register team commands
	rootCmd.AddCommand(createTeamCmd)
	rootCmd.AddCommand(addTeamDepCmd)

	// Add subcommands to their parent commands
	metricsCmd.AddCommand(metricsShowCmd)
	metricsCmd.AddCommand(metricsClearCmd)
	
	monitorCmd.AddCommand(monitorShowCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	
	// Add commands to the root command
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(tracesCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Helper function to display container stats
func displayContainerStats(stat *monitor.ContainerStats) {
	fmt.Printf("  CPU: %.2f%%\n", stat.CPUUsage)
	fmt.Printf("  Memory: %.2f%% (%.2f MB / %.2f MB)\n", 
		stat.MemoryPercent, 
		float64(stat.MemoryUsage)/(1024*1024), 
		float64(stat.MemoryLimit)/(1024*1024))
	fmt.Printf("  Disk: Read %.2f MB, Write %.2f MB\n", 
		float64(stat.DiskRead)/(1024*1024), 
		float64(stat.DiskWrite)/(1024*1024))
	fmt.Printf("  Network: Rx %.2f MB, Tx %.2f MB\n", 
		float64(stat.NetRx)/(1024*1024), 
		float64(stat.NetTx)/(1024*1024))
	fmt.Printf("  Last Update: %s\n", stat.Timestamp.Format(time.RFC3339))
} 