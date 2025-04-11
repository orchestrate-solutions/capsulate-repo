package main

import (
	"fmt"
	"os"

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
			
			// Initialize agent manager
			manager := agent.NewManager()
			
			// Configure agent
			config := agent.AgentConfig{
				ID:              agentID,
				DependencyLevel: dependencyLevel,
				OverrideDeps:    overrideDepsList,
				UseOverlay:      useOverlay,
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
	rootCmd.AddCommand(destroyCmd)
	
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 