package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/yourusername/dcode/internal/agent"
	"github.com/yourusername/dcode/internal/config"
	"github.com/yourusername/dcode/internal/provider"
	"github.com/yourusername/dcode/internal/server"
	"github.com/yourusername/dcode/internal/session"
	"github.com/yourusername/dcode/internal/tool"
	"github.com/yourusername/dcode/internal/tui"
)

var (
	version = "2.0.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dcode",
		Short: "DCode - AI-powered coding agent",
		Long: `DCode is an advanced AI coding agent that runs in your terminal.
It can read, write, and search code, execute commands, and help you
with software engineering tasks.`,
		RunE:          runTUI,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Flags
	rootCmd.PersistentFlags().StringP("provider", "p", "", "AI provider (anthropic, openai, copilot, google, groq, openrouter, xai, deepseek, etc.)")
	rootCmd.PersistentFlags().StringP("model", "m", "", "Model to use (provider/model format supported)")
	rootCmd.PersistentFlags().StringP("agent", "a", "", "Agent to use (coder, planner, explorer, general)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Sub-commands
	rootCmd.AddCommand(
		runCmd(),
		serveCmd(),
		authCmd(),
		sessionCmd(),
		modelsCmd(),
		agentCmd(),
		toolsCmd(),
		exportCmd(),
		importCmd(),
		configCmd(),
		mcpCmd(),
		statsCmd(),
		debugCmd(),
		upgradeCmd(),
		uninstallCmd(),
		completionCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUI is the default command - starts the TUI
func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	applyFlags(cmd, cfg)

	// Initialize session store
	store, err := session.NewStore(cfg.SessionDir)
	if err != nil {
		return fmt.Errorf("failed to init session store: %w", err)
	}

	// Get agent name
	agentName := cfg.DefaultAgent
	if agentName == "" {
		agentName = "coder"
	}

	// Create and run TUI — provider is initialized asynchronously inside Init()
	model := tui.New(store, nil, cfg, agentName, cfg.GetDefaultModel(cfg.Provider), cfg.Provider)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// run command
// ---------------------------------------------------------------------------

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [message...]",
		Short: "Run a non-interactive session with a prompt",
		Long:  "Execute a prompt without the TUI. Useful for scripting and CI/CD.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			applyFlags(cmd, cfg)

			prov, err := initProvider(cfg)
			if err != nil {
				return err
			}

			store, err := session.NewStore(cfg.SessionDir)
			if err != nil {
				return err
			}

			registry := tool.GetRegistry()
			agentName := cfg.DefaultAgent
			if agentName == "" {
				agentName = "coder"
			}
			ag := agent.GetAgent(agentName, cfg)

			engine := session.NewPromptEngine(store, prov, cfg, ag, registry)

			// Stream output to stdout
			engine.OnStream(func(event session.StreamEvent) {
				switch event.Type {
				case "text":
					fmt.Print(event.Content)
				case "tool_start":
					fmt.Fprintf(os.Stderr, "\n> %s\n", event.ToolName)
				case "tool_end":
					// nothing
				case "error":
					fmt.Fprintf(os.Stderr, "\nError: %s\n", event.Content)
				case "done":
					fmt.Println()
				}
			})

			// Create session
			sess, err := store.Create(agentName, cfg.GetDefaultModel(cfg.Provider), cfg.Provider)
			if err != nil {
				return err
			}

			message := strings.Join(args, " ")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle Ctrl+C
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			return engine.Run(ctx, sess.ID, message)
		},
	}
	return cmd
}

// ---------------------------------------------------------------------------
// serve command
// ---------------------------------------------------------------------------

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		Long:  "Start a headless HTTP API server for programmatic access to DCode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			applyFlags(cmd, cfg)
			cfg.Server.Enabled = true

			store, err := session.NewStore(cfg.SessionDir)
			if err != nil {
				return err
			}

			registry := tool.GetRegistry()
			srv := server.New(cfg, store, registry)

			// Handle shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println("\nShutting down server...")
				srv.Stop()
				cancel()
			}()

			_ = ctx
			return srv.Start()
		},
	}
	cmd.Flags().IntP("port", "P", 4096, "Port to listen on")
	return cmd
}

// ---------------------------------------------------------------------------
// auth command (replaces login/logout, adds list)
// ---------------------------------------------------------------------------

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Aliases: []string{"login"},
		Short:   "Manage API credentials",
		Long:    "Log in, log out, and list configured API provider credentials.",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Set up API key authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Login()
		},
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored API credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Logout()
		},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured credentials and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := config.LoadCredentials()
			if err != nil {
				return fmt.Errorf("failed to load credentials: %w", err)
			}

			cfg, _ := config.Load()

			type provEntry struct {
				name   string
				envVar string
				cred   string
			}

			entries := []provEntry{
				{"anthropic", "ANTHROPIC_API_KEY", creds.AnthropicAPIKey},
				{"openai", "OPENAI_API_KEY", creds.OpenAIAPIKey},
				{"copilot", "GITHUB_TOKEN", creds.GitHubToken},
				{"google", "GOOGLE_API_KEY", creds.GoogleAPIKey},
				{"groq", "GROQ_API_KEY", creds.GroqAPIKey},
				{"openrouter", "OPENROUTER_API_KEY", creds.OpenRouterKey},
				{"xai", "XAI_API_KEY", creds.XAIAPIKey},
				{"deepseek", "DEEPSEEK_API_KEY", creds.DeepSeekAPIKey},
				{"mistral", "MISTRAL_API_KEY", creds.MistralAPIKey},
				{"deepinfra", "DEEPINFRA_API_KEY", creds.DeepInfraAPIKey},
				{"cerebras", "CEREBRAS_API_KEY", creds.CerebrasAPIKey},
				{"together", "TOGETHER_API_KEY", creds.TogetherAPIKey},
				{"cohere", "COHERE_API_KEY", creds.CohereAPIKey},
				{"perplexity", "PERPLEXITY_API_KEY", creds.PerplexityAPIKey},
				{"azure", "AZURE_OPENAI_API_KEY", creds.AzureAPIKey},
				{"bedrock", "AWS_ACCESS_KEY_ID", ""},
				{"gitlab", "GITLAB_TOKEN", creds.GitLabToken},
				{"cloudflare", "CLOUDFLARE_API_TOKEN", creds.CloudflareAPIToken},
				{"replicate", "REPLICATE_API_TOKEN", creds.ReplicateAPIToken},
			}

			fmt.Printf("%-22s %-12s %-10s\n", "Provider", "Source", "Status")
			fmt.Println(strings.Repeat("-", 50))

			for _, e := range entries {
				source := ""
				status := "not configured"

				// Check env var
				if os.Getenv(e.envVar) != "" {
					source = "env"
					status = "active"
				}

				// Check stored credentials
				if source == "" && e.cred != "" {
					source = "credentials"
					status = "active"
				}

				// Check config
				if source == "" && cfg != nil && cfg.GetAPIKey(e.name) != "" {
					source = "config"
					status = "active"
				}

				// Special: copilot gh CLI
				if source == "" && e.name == "copilot" {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					if out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output(); err == nil && strings.TrimSpace(string(out)) != "" {
						source = "gh-cli"
						status = "active"
					}
					cancel()
				}

				if status == "active" {
					fmt.Printf("%-22s %-12s %-10s\n", e.name, source, status)
				}
			}

			return nil
		},
	}

	copilotCmd := &cobra.Command{
		Use:   "copilot",
		Short: "Authenticate with GitHub Copilot via OAuth device flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return provider.CopilotLogin()
		},
	}

	cmd.AddCommand(loginCmd, logoutCmd, listCmd, copilotCmd)

	// Default to login if no subcommand given
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return config.Login()
	}

	return cmd
}

// ---------------------------------------------------------------------------
// session command
// ---------------------------------------------------------------------------

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage conversation sessions",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all sessions",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, _ := config.Load()
				store, err := session.NewStore(cfg.SessionDir)
				if err != nil {
					return err
				}

				sessions := store.List()
				if len(sessions) == 0 {
					fmt.Println("No sessions found.")
					return nil
				}

				format, _ := cmd.Flags().GetString("format")
				if format == "json" {
					data, _ := json.MarshalIndent(sessions, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				fmt.Printf("%-10s %-30s %-10s %-8s %-15s\n", "ID", "Title", "Agent", "Msgs", "Updated")
				fmt.Println(strings.Repeat("-", 80))
				for _, s := range sessions {
					title := s.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					fmt.Printf("%-10s %-30s %-10s %-8d %-15s\n",
						s.ID, title, s.Agent, len(s.Messages),
						s.UpdatedAt.Format("Jan 02 15:04"))
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "delete [id]",
			Short: "Delete a session",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, _ := config.Load()
				store, err := session.NewStore(cfg.SessionDir)
				if err != nil {
					return err
				}
				if err := store.Delete(args[0]); err != nil {
					return err
				}
				fmt.Printf("Session %s deleted.\n", args[0])
				return nil
			},
		},
	)

	// Add format flag to list
	cmd.Commands()[0].Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

// ---------------------------------------------------------------------------
// models command
// ---------------------------------------------------------------------------

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models [provider]",
		Short: "List available AI models",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filterProvider := ""
			if len(args) > 0 {
				filterProvider = args[0]
			}

			fmt.Printf("%-22s %-40s %-10s %-10s %-6s\n", "Provider", "Model", "Context", "Output", "Tools")
			fmt.Println(strings.Repeat("-", 95))

			for prov, info := range config.DefaultModels {
				if filterProvider != "" && prov != filterProvider {
					continue
				}
				toolSupport := "yes"
				if !info.SupportsTools {
					toolSupport = "no"
				}
				fmt.Printf("%-22s %-40s %-10s %-10s %-6s\n",
					prov, info.ID,
					formatContextSize(info.ContextWindow),
					formatContextSize(info.MaxOutput),
					toolSupport)
			}
			return nil
		},
	}
	return cmd
}

// ---------------------------------------------------------------------------
// agent command (parent with list and create subcommands)
// ---------------------------------------------------------------------------

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents"},
		Short:   "Manage agents",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all available agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			allAgents := agent.ListAgents(cfg)

			fmt.Printf("%-14s %-10s %-6s %-8s %s\n", "Name", "Mode", "Steps", "Hidden", "Description")
			fmt.Println(strings.Repeat("-", 80))
			for _, ag := range allAgents {
				hidden := ""
				if ag.Hidden {
					hidden = "yes"
				}
				fmt.Printf("%-14s %-10s %-6d %-8s %s\n",
					ag.Name, ag.Mode, ag.Steps, hidden, ag.Description)
			}
			return nil
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent from a markdown file",
		Long: `Create a new agent by generating a markdown file with YAML frontmatter.
The file is placed in .dcode/agents/ directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			desc, _ := cmd.Flags().GetString("description")
			mode, _ := cmd.Flags().GetString("mode")
			if mode == "" {
				mode = "subagent"
			}

			// Create .dcode/agents/ directory
			dir := filepath.Join(".dcode", "agents")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Generate markdown content
			content := fmt.Sprintf(`---
description: %s
mode: %s
---

You are a %s agent. %s
`, desc, mode, name, desc)

			filePath := filepath.Join(dir, name+".md")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write agent file: %w", err)
			}

			fmt.Printf("Agent '%s' created at %s\n", name, filePath)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Agent name (required)")
	createCmd.Flags().String("description", "", "Agent description")
	createCmd.Flags().String("mode", "subagent", "Agent mode (primary, subagent, all)")
	_ = createCmd.MarkFlagRequired("name")

	cmd.AddCommand(listCmd, createCmd)

	// Default to list
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return listCmd.RunE(cmd, args)
	}

	return cmd
}

// ---------------------------------------------------------------------------
// tools command
// ---------------------------------------------------------------------------

func toolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List available tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := tool.GetRegistry()
			tools := registry.GetAll()
			fmt.Printf("%-18s %s\n", "Name", "Description")
			fmt.Println(strings.Repeat("-", 70))
			for name, t := range tools {
				desc := t.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				fmt.Printf("%-18s %s\n", name, desc)
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// export / import commands
// ---------------------------------------------------------------------------

func exportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [session-id]",
		Short: "Export a session as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			store, err := session.NewStore(cfg.SessionDir)
			if err != nil {
				return err
			}
			data, err := store.Export(args[0])
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import a session from JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			cfg, _ := config.Load()
			store, err := session.NewStore(cfg.SessionDir)
			if err != nil {
				return err
			}
			sess, err := store.Import(data)
			if err != nil {
				return err
			}
			fmt.Printf("Imported session: %s (%s)\n", sess.ID, sess.Title)
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// config command
// ---------------------------------------------------------------------------

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or update configuration",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show current configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				data, _ := json.MarshalIndent(map[string]interface{}{
					"provider":            cfg.Provider,
					"model":               cfg.GetDefaultModel(cfg.Provider),
					"default_agent":       cfg.DefaultAgent,
					"streaming":           cfg.Streaming,
					"max_tokens":          cfg.MaxTokens,
					"theme":               cfg.Theme,
					"session_dir":         cfg.SessionDir,
					"snapshot":            cfg.Snapshot,
					"compaction":          cfg.Compaction,
					"username":            cfg.Username,
					"available_providers": cfg.ListAvailableProviders(),
					"config_directories":  cfg.Directories(),
				}, "", "  ")
				fmt.Println(string(data))
				return nil
			},
		},
		&cobra.Command{
			Use:   "set [key] [value]",
			Short: "Set a configuration value",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				switch args[0] {
				case "provider":
					cfg.Provider = args[1]
				case "model":
					cfg.Model = args[1]
				case "agent", "default_agent":
					cfg.DefaultAgent = args[1]
				case "theme":
					cfg.Theme = args[1]
				case "username":
					cfg.Username = args[1]
				case "small_model":
					cfg.SmallModel = args[1]
				default:
					return fmt.Errorf("unknown config key: %s\nSupported keys: provider, model, agent, theme, username, small_model", args[0])
				}
				configDir := config.GetConfigDir()
				return cfg.SaveConfig(filepath.Join(configDir, "dcode.json"))
			},
		},
	)

	// Default to show
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Commands()[0].RunE(cmd, args)
	}

	return cmd
}

// ---------------------------------------------------------------------------
// mcp command – manage MCP servers
// ---------------------------------------------------------------------------

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) servers",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured MCP servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(cfg.MCP) == 0 {
				fmt.Println("No MCP servers configured.")
				fmt.Println("\nAdd one with: dcode mcp add")
				return nil
			}

			fmt.Printf("%-20s %-10s %-40s %-8s\n", "Name", "Type", "Target", "Enabled")
			fmt.Println(strings.Repeat("-", 80))
			for name, mcp := range cfg.MCP {
				target := ""
				if mcp.Type == "local" {
					target = strings.Join(mcp.Command, " ")
				} else {
					target = mcp.URL
				}
				if len(target) > 38 {
					target = target[:35] + "..."
				}
				enabled := "yes"
				if mcp.Enabled != nil && !*mcp.Enabled {
					enabled = "no"
				}
				fmt.Printf("%-20s %-10s %-40s %-8s\n", name, mcp.Type, target, enabled)
			}
			return nil
		},
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add an MCP server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			mcpType, _ := cmd.Flags().GetString("type")
			command, _ := cmd.Flags().GetString("command")
			url, _ := cmd.Flags().GetString("url")

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			mcpCfg := config.MCPConfig{Type: mcpType}
			if mcpType == "local" {
				if command == "" {
					return fmt.Errorf("--command is required for local MCP servers")
				}
				mcpCfg.Command = strings.Fields(command)
			} else {
				if url == "" {
					return fmt.Errorf("--url is required for remote MCP servers")
				}
				mcpCfg.URL = url
			}

			// Load config, add MCP, save
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.MCP == nil {
				cfg.MCP = make(map[string]config.MCPConfig)
			}
			cfg.MCP[name] = mcpCfg

			configDir := config.GetConfigDir()
			if err := cfg.SaveConfig(filepath.Join(configDir, "dcode.json")); err != nil {
				return err
			}

			fmt.Printf("MCP server '%s' added (%s)\n", name, mcpType)
			return nil
		},
	}
	addCmd.Flags().String("name", "", "Name for the MCP server (required)")
	addCmd.Flags().String("type", "local", "Type: local or remote")
	addCmd.Flags().String("command", "", "Command for local MCP server")
	addCmd.Flags().String("url", "", "URL for remote MCP server")
	_ = addCmd.MarkFlagRequired("name")

	removeCmd := &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Short:   "Remove an MCP server configuration",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.MCP[args[0]]; !ok {
				return fmt.Errorf("MCP server '%s' not found", args[0])
			}
			delete(cfg.MCP, args[0])

			configDir := config.GetConfigDir()
			if err := cfg.SaveConfig(filepath.Join(configDir, "dcode.json")); err != nil {
				return err
			}

			fmt.Printf("MCP server '%s' removed\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd)

	// Default to list
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return listCmd.RunE(cmd, args)
	}

	return cmd
}

// ---------------------------------------------------------------------------
// stats command – session statistics
// ---------------------------------------------------------------------------

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show session usage statistics",
		Long:  "Display token usage, cost, and tool usage statistics across sessions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			store, err := session.NewStore(cfg.SessionDir)
			if err != nil {
				return err
			}

			sessions := store.List()
			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}

			totalSessions := len(sessions)
			totalMessages := 0
			totalToolCalls := 0
			agentCounts := make(map[string]int)
			providerCounts := make(map[string]int)
			modelCounts := make(map[string]int)

			for _, s := range sessions {
				totalMessages += len(s.Messages)
				agentCounts[s.Agent]++
				providerCounts[s.Provider]++
				modelCounts[s.Model]++

				for _, m := range s.Messages {
					for _, p := range m.Parts {
						if p.Type == "tool_use" {
							totalToolCalls++
						}
					}
				}
			}

			fmt.Println("=== DCode Usage Statistics ===")
			fmt.Println()
			fmt.Printf("  Sessions:    %d\n", totalSessions)
			fmt.Printf("  Messages:    %d\n", totalMessages)
			fmt.Printf("  Tool calls:  %d\n", totalToolCalls)
			fmt.Println()

			if len(providerCounts) > 0 {
				fmt.Println("Provider usage:")
				for prov, count := range providerCounts {
					fmt.Printf("  %-20s %d sessions\n", prov, count)
				}
				fmt.Println()
			}

			if len(modelCounts) > 0 {
				fmt.Println("Model usage:")
				for model, count := range modelCounts {
					fmt.Printf("  %-40s %d sessions\n", model, count)
				}
				fmt.Println()
			}

			if len(agentCounts) > 0 {
				fmt.Println("Agent usage:")
				for ag, count := range agentCounts {
					fmt.Printf("  %-20s %d sessions\n", ag, count)
				}
			}

			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// debug command – debugging / troubleshooting
// ---------------------------------------------------------------------------

func debugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debugging and troubleshooting tools",
	}

	configDebugCmd := &cobra.Command{
		Use:   "config",
		Short: "Show the fully resolved configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	pathsCmd := &cobra.Command{
		Use:   "paths",
		Short: "Show dcode paths (data, config, cache)",
		Run: func(cmd *cobra.Command, args []string) {
			home, _ := os.UserHomeDir()
			configDir := config.GetConfigDir()
			projectDir := config.GetProjectDir()

			fmt.Println("Paths:")
			fmt.Printf("  Config:     %s\n", configDir)
			fmt.Printf("  Data:       %s\n", filepath.Join(home, ".local", "share", "dcode"))
			fmt.Printf("  Cache:      %s\n", filepath.Join(home, ".cache", "dcode"))
			fmt.Printf("  Sessions:   %s\n", filepath.Join(configDir, "sessions"))
			fmt.Printf("  Credentials:%s\n", filepath.Join(configDir, "credentials.json"))
			fmt.Printf("  Project:    %s\n", projectDir)
			fmt.Println()

			dirs, _ := filepath.Glob(filepath.Join(projectDir, ".dcode"))
			if len(dirs) > 0 {
				fmt.Println("Project config directories:")
				for _, d := range dirs {
					fmt.Printf("  %s\n", d)
				}
			}
		},
	}

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Show relevant environment variables",
		Run: func(cmd *cobra.Command, args []string) {
			envVars := []string{
				"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GITHUB_TOKEN",
				"GOOGLE_API_KEY", "GEMINI_API_KEY", "GROQ_API_KEY",
				"OPENROUTER_API_KEY", "XAI_API_KEY", "DEEPSEEK_API_KEY",
				"MISTRAL_API_KEY", "DEEPINFRA_API_KEY", "CEREBRAS_API_KEY",
				"TOGETHER_API_KEY", "COHERE_API_KEY", "PERPLEXITY_API_KEY",
				"AZURE_OPENAI_API_KEY", "AWS_ACCESS_KEY_ID",
				"GITLAB_TOKEN", "CLOUDFLARE_API_TOKEN", "REPLICATE_API_TOKEN",
				"DCODE_CONFIG", "DCODE_CONFIG_DIR", "DCODE_CONFIG_CONTENT",
				"DCODE_PERMISSION", "DCODE_DISABLE_AUTOCOMPACT", "DCODE_DISABLE_PRUNE",
				"DCODE_DISABLE_PROJECT_CONFIG",
			}

			fmt.Println("Environment variables:")
			for _, ev := range envVars {
				val := os.Getenv(ev)
				if val != "" {
					// Mask API keys
					if strings.Contains(ev, "KEY") || strings.Contains(ev, "TOKEN") || strings.Contains(ev, "SECRET") {
						if len(val) > 8 {
							val = val[:4] + "..." + val[len(val)-4:]
						} else {
							val = "****"
						}
					}
					fmt.Printf("  %-30s %s\n", ev, val)
				}
			}
		},
	}

	cmd.AddCommand(configDebugCmd, pathsCmd, envCmd)
	return cmd
}

// ---------------------------------------------------------------------------
// upgrade command – self-update
// ---------------------------------------------------------------------------

func upgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [version]",
		Short: "Upgrade dcode to the latest version",
		Long:  "Download and install the latest version of dcode, or a specific version.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "latest"
			if len(args) > 0 {
				target = args[0]
			}

			fmt.Printf("Current version: %s (%s)\n", version, commit)
			fmt.Printf("Upgrade target:  %s\n\n", target)

			// Detect installation method
			method := detectInstallMethod()
			if method == "" {
				return fmt.Errorf("could not detect installation method. Please upgrade manually:\n  go install github.com/yourusername/dcode/cmd/dcode@%s", target)
			}

			fmt.Printf("Detected installation method: %s\n", method)

			switch method {
			case "go":
				installTarget := target
				if installTarget == "latest" {
					installTarget = "latest"
				} else if !strings.HasPrefix(installTarget, "v") {
					installTarget = "v" + installTarget
				}
				fmt.Printf("Running: go install github.com/yourusername/dcode/cmd/dcode@%s\n", installTarget)
				out, err := exec.Command("go", "install", "github.com/yourusername/dcode/cmd/dcode@"+installTarget).CombinedOutput()
				if err != nil {
					return fmt.Errorf("upgrade failed: %s\n%s", err, string(out))
				}
				fmt.Println("Upgrade successful!")
			default:
				return fmt.Errorf("automatic upgrade not supported for installation method '%s'. Please upgrade manually", method)
			}

			return nil
		},
	}
}

func detectInstallMethod() string {
	// Check if installed via go install
	exePath, err := os.Executable()
	if err == nil {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home, _ := os.UserHomeDir()
			gopath = filepath.Join(home, "go")
		}
		if strings.HasPrefix(exePath, filepath.Join(gopath, "bin")) {
			return "go"
		}
	}
	return "go" // default to go install
}

// ---------------------------------------------------------------------------
// uninstall command
// ---------------------------------------------------------------------------

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall dcode and remove all related files",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			keepConfig, _ := cmd.Flags().GetBool("keep-config")
			keepData, _ := cmd.Flags().GetBool("keep-data")

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			paths := []struct {
				path  string
				label string
				skip  bool
			}{
				{filepath.Join(home, ".config", "dcode"), "Configuration", keepConfig},
				{filepath.Join(home, ".local", "share", "dcode"), "Data", keepData},
				{filepath.Join(home, ".cache", "dcode"), "Cache", false},
			}

			if dryRun {
				fmt.Println("Dry run - would remove:")
				for _, p := range paths {
					if p.skip {
						fmt.Printf("  [KEEP] %s (%s)\n", p.path, p.label)
					} else {
						fmt.Printf("  [REMOVE] %s (%s)\n", p.path, p.label)
					}
				}
				return nil
			}

			for _, p := range paths {
				if p.skip {
					fmt.Printf("Keeping %s: %s\n", p.label, p.path)
					continue
				}
				if _, err := os.Stat(p.path); err == nil {
					if err := os.RemoveAll(p.path); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", p.path, err)
					} else {
						fmt.Printf("Removed %s: %s\n", p.label, p.path)
					}
				}
			}

			// Remove binary
			exePath, err := os.Executable()
			if err == nil {
				fmt.Printf("\nTo complete uninstall, remove the binary:\n  rm %s\n", exePath)
			}

			fmt.Println("\nDCode has been uninstalled.")
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be removed without removing")
	cmd.Flags().BoolP("keep-config", "c", false, "Keep configuration files")
	cmd.Flags().BoolP("keep-data", "d", false, "Keep session data")
	return cmd
}

// ---------------------------------------------------------------------------
// completion command
// ---------------------------------------------------------------------------

func completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", args[0])
			}
		},
	}
}

// ---------------------------------------------------------------------------
// version command
// ---------------------------------------------------------------------------

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dcode version %s (%s)\n", version, commit)
			fmt.Printf("go version %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func applyFlags(cmd *cobra.Command, cfg *config.Config) {
	if p, _ := cmd.Flags().GetString("provider"); p != "" {
		cfg.Provider = p
	}
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		// Support provider/model format
		if strings.Contains(m, "/") {
			parts := strings.SplitN(m, "/", 2)
			cfg.Provider = parts[0]
			cfg.Model = parts[1]
		} else {
			cfg.Model = m
		}
	}
	if a, _ := cmd.Flags().GetString("agent"); a != "" {
		cfg.DefaultAgent = a
	}
	if v, _ := cmd.Flags().GetBool("verbose"); v {
		cfg.Verbose = true
	}
}

func initProvider(cfg *config.Config) (provider.Provider, error) {
	apiKey, err := config.GetAPIKeyWithFallback(cfg.Provider, cfg)
	if err != nil {
		return nil, fmt.Errorf("no API key for %s. Run 'dcode auth login' to set up authentication.\n\nOr set environment variable:\n  export %s=your-api-key",
			cfg.Provider, getEnvVarName(cfg.Provider))
	}

	prov, err := provider.CreateProvider(cfg.Provider, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", cfg.Provider, err)
	}

	return prov, nil
}

func getEnvVarName(providerName string) string {
	envVars := map[string]string{
		"anthropic":             "ANTHROPIC_API_KEY",
		"openai":                "OPENAI_API_KEY",
		"copilot":               "GITHUB_TOKEN",
		"google":                "GOOGLE_API_KEY",
		"groq":                  "GROQ_API_KEY",
		"openrouter":            "OPENROUTER_API_KEY",
		"xai":                   "XAI_API_KEY",
		"deepseek":              "DEEPSEEK_API_KEY",
		"mistral":               "MISTRAL_API_KEY",
		"deepinfra":             "DEEPINFRA_API_KEY",
		"cerebras":              "CEREBRAS_API_KEY",
		"together":              "TOGETHER_API_KEY",
		"cohere":                "COHERE_API_KEY",
		"perplexity":            "PERPLEXITY_API_KEY",
		"azure":                 "AZURE_OPENAI_API_KEY",
		"bedrock":               "AWS_ACCESS_KEY_ID",
		"google-vertex":         "GOOGLE_CLOUD_PROJECT",
		"gitlab":                "GITLAB_TOKEN",
		"cloudflare-workers-ai": "CLOUDFLARE_API_TOKEN",
		"replicate":             "REPLICATE_API_TOKEN",
	}
	if v, ok := envVars[providerName]; ok {
		return v
	}
	return strings.ToUpper(providerName) + "_API_KEY"
}

func formatContextSize(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	return fmt.Sprintf("%dK", tokens/1000)
}
