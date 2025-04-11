package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/capsulate-repo/pkg/agent"
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
			dependencyLevel, _ := cmd.Flags().GetString("dependency-level")
			overrideDepsList, _ := cmd.Flags().GetStringSlice("override-deps")
			useOverlay, _ := cmd.Flags().GetBool("use-overlay")
			
			// Get repository options
			repoURL, _ := cmd.Flags().GetString("repo")
			branch, _ := cmd.Flags().GetString("branch")
			depth, _ := cmd.Flags().GetInt("depth")
			gitConfigStr, _ := cmd.Flags().GetStringSlice("git-config")
			
			// Parse git config into map
			gitConfig := make(map[string]string)
			for _, cfg := range gitConfigStr {
				parts := strings.SplitN(cfg, "=", 2)
				if len(parts) == 2 {
					gitConfig[parts[0]] = parts[1]
				}
			}
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Configure agent
			config := agent.AgentConfig{
				ID:              agentID,
				DependencyLevel: dependencyLevel,
				OverrideDeps:    overrideDepsList,
				UseOverlay:      useOverlay,
				RepoURL:         repoURL,
				Branch:          branch,
				Depth:           depth,
				GitConfig:       gitConfig,
			}
			
			// Create the agent
			if err := manager.Create(config); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Agent '%s' created successfully\n", agentID)
		},
	}
	
	// Add flags for create command
	createCmd.Flags().String("dependency-level", "core", "Dependency isolation level (core, team, container)")
	createCmd.Flags().StringSlice("override-deps", []string{}, "List of dependencies to override")
	createCmd.Flags().Bool("use-overlay", false, "Use overlay filesystem")
	// Add repository flags
	createCmd.Flags().String("repo", "", "Git repository URL to clone")
	createCmd.Flags().String("branch", "", "Branch to checkout")
	createCmd.Flags().Int("depth", 0, "Depth for shallow clones (0 for full clone)")
	createCmd.Flags().StringSlice("git-config", []string{}, "Git configurations to apply (format: key=value)")
	
	// Add exec command
	execCmd := &cobra.Command{
		Use:   "exec [agent-id] [command]",
		Short: "Execute a command in the agent container",
		Long:  `Execute a command in the specified agent container.`,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			command := args[1]
			
			// For multiple arguments, combine them into a single command
			if len(args) > 2 {
				for i := 2; i < len(args); i++ {
					command += " " + args[i]
				}
			}
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Execute the command
			output, err := manager.Exec(agentID, command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Print(output)
		},
	}
	
	// Add status command
	statusCmd := &cobra.Command{
		Use:   "status [agent-id]",
		Short: "Get Git status from an agent container",
		Long:  `Retrieve and display the Git status of the repository in the specified agent container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Get Git status
			status, err := manager.GetGitStatus(agentID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting Git status: %v\n", err)
				os.Exit(1)
			}
			
			// Display Git status in a formatted way
			fmt.Printf("Branch: %s\n", status.Branch)
			fmt.Printf("Commit: %.8s\n", status.CurrentCommit)
			fmt.Printf("Ahead: %d, Behind: %d\n", status.AheadCount, status.BehindCount)
			
			if len(status.ModifiedFiles) > 0 {
				fmt.Println("\nModified files:")
				for _, file := range status.ModifiedFiles {
					fmt.Printf("  - %s\n", file)
				}
			}
			
			if len(status.UntrackedFiles) > 0 {
				fmt.Println("\nUntracked files:")
				for _, file := range status.UntrackedFiles {
					fmt.Printf("  - %s\n", file)
				}
			}
		},
	}
	
	// Add branch command
	branchCmd := &cobra.Command{
		Use:   "branch [agent-id] [branch-name]",
		Short: "Create a new Git branch in the agent container",
		Long:  `Create a new Git branch in the specified agent container.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			branchName := args[1]
			checkout, _ := cmd.Flags().GetBool("checkout")
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Create the branch
			if err := manager.CreateBranch(agentID, branchName, checkout); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating branch: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Branch '%s' created successfully\n", branchName)
			if checkout {
				fmt.Printf("Switched to branch '%s'\n", branchName)
			}
		},
	}
	
	// Add flags for branch command
	branchCmd.Flags().Bool("checkout", true, "Checkout the branch after creation")
	
	// Add checkout command
	checkoutCmd := &cobra.Command{
		Use:   "checkout [agent-id] [branch-name]",
		Short: "Checkout a Git branch in the agent container",
		Long:  `Checkout a Git branch in the specified agent container.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			branchName := args[1]
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Checkout the branch
			if err := manager.CheckoutBranch(agentID, branchName); err != nil {
				fmt.Fprintf(os.Stderr, "Error checking out branch: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Switched to branch '%s'\n", branchName)
		},
	}
	
	// Add destroy command
	destroyCmd := &cobra.Command{
		Use:   "destroy [agent-id]",
		Short: "Destroy an agent container",
		Long:  `Destroy the specified agent container.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agentID := args[0]
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Destroy the agent
			if err := manager.Destroy(agentID); err != nil {
				fmt.Fprintf(os.Stderr, "Error destroying agent: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Agent '%s' destroyed successfully\n", agentID)
		},
	}
	
	// Add commands to root
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(destroyCmd)
	
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 