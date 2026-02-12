package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/yourusername/dcode/internal/agent"
	"github.com/yourusername/dcode/internal/config"
	"github.com/yourusername/dcode/internal/provider"
	"github.com/yourusername/dcode/internal/session"
	"github.com/yourusername/dcode/internal/server"
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
	rootCmd.PersistentFlags().StringP("provider", "p", "", "AI provider (anthropic, openai, copilot, google, groq, openrouter)")
	rootCmd.PersistentFlags().StringP("model", "m", "", "Model to use")
	rootCmd.PersistentFlags().StringP("agent", "a", "", "Agent to use (coder, planner, explorer, researcher)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Sub-commands
	rootCmd.AddCommand(
		runCmd(),
		serveCmd(),
		loginCmd(),
		logoutCmd(),
		sessionCmd(),
		modelsCmd(),
		agentsCmd(),
		toolsCmd(),
		exportCmd(),
		importCmd(),
		configCmd(),
		copilotLoginCmd(),
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

	// Initialize provider
	prov, err := initProvider(cfg)
	if err != nil {
		return err
	}

	// Initialize session store
	store, err := session.NewStore(cfg.SessionDir)
	if err != nil {
		return fmt.Errorf("failed to init session store: %w", err)
	}

	// Initialize tool registry
	registry := tool.GetRegistry()

	// Get agent
	agentName := cfg.DefaultAgent
	if agentName == "" {
		agentName = "coder"
	}
	ag := agent.GetAgent(agentName, cfg)

	// Create prompt engine
	engine := session.NewPromptEngine(store, prov, cfg, ag, registry)

	// Create and run TUI
	model := tui.New(store, engine, cfg, agentName, cfg.GetDefaultModel(cfg.Provider), cfg.Provider)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// runCmd runs a non-interactive prompt
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
					fmt.Fprintf(os.Stderr, "\n⚡ %s\n", event.ToolName)
				case "tool_end":
					// nothing
				case "error":
					fmt.Fprintf(os.Stderr, "\n❌ %s\n", event.Content)
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

// serveCmd starts the HTTP API server
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

// loginCmd handles authentication setup
func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Set up API key authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Login()
		},
	}
}

// copilotLoginCmd authenticates with GitHub Copilot via OAuth device flow
func copilotLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copilot-login",
		Short: "Authenticate with GitHub Copilot via OAuth device flow",
		Long:  "Performs the GitHub OAuth device flow to get a token for GitHub Copilot API access.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return provider.CopilotLogin()
		},
	}
}

// logoutCmd removes stored credentials
func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored API credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Logout()
		},
	}
}

// sessionCmd manages sessions
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

				fmt.Printf("%-10s %-30s %-10s %-8s %-15s\n", "ID", "Title", "Agent", "Msgs", "Updated")
				fmt.Println(strings.Repeat("─", 80))
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

	return cmd
}

// modelsCmd lists available models
func modelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%-12s %-35s %-15s %-10s\n", "Provider", "Model", "Context", "Output")
			fmt.Println(strings.Repeat("─", 80))

			for prov, info := range config.DefaultModels {
				fmt.Printf("%-12s %-35s %-15s %-10s\n",
					prov, info.ID,
					formatContextSize(info.ContextWindow),
					formatContextSize(info.MaxOutput))
			}
			return nil
		},
	}
}

// agentsCmd lists available agents
func agentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List available agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			agents := agent.BuiltinAgents()
			fmt.Printf("%-12s %-8s %-5s %s\n", "Name", "Mode", "Steps", "Description")
			fmt.Println(strings.Repeat("─", 70))
			for name, ag := range agents {
				fmt.Printf("%-12s %-8s %-5d %s\n", name, ag.Mode, ag.Steps, ag.Description)
			}
			return nil
		},
	}
}

// toolsCmd lists available tools
func toolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List available tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := tool.GetRegistry()
			tools := registry.GetAll()
			fmt.Printf("%-12s %s\n", "Name", "Description")
			fmt.Println(strings.Repeat("─", 70))
			for name, t := range tools {
				desc := t.Description
				if len(desc) > 55 {
					desc = desc[:52] + "..."
				}
				fmt.Printf("%-12s %s\n", name, desc)
			}
			return nil
		},
	}
}

// exportCmd exports a session
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

// importCmd imports a session
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

// configCmd manages configuration
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
					"provider":      cfg.Provider,
					"model":         cfg.GetDefaultModel(cfg.Provider),
					"default_agent": cfg.DefaultAgent,
					"streaming":     cfg.Streaming,
					"max_tokens":    cfg.MaxTokens,
					"theme":         cfg.Theme,
					"session_dir":   cfg.SessionDir,
					"available":     cfg.ListAvailableProviders(),
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
				case "agent":
					cfg.DefaultAgent = args[1]
				case "theme":
					cfg.Theme = args[1]
				default:
					return fmt.Errorf("unknown config key: %s", args[0])
				}
				configDir := config.GetConfigDir()
				return cfg.SaveConfig(configDir + "/dcode.json")
			},
		},
	)

	return cmd
}

// completionCmd generates shell completion
func completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
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
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}

// versionCmd shows the version
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dcode version %s (%s)\n", version, commit)
		},
	}
}

// Helper functions

func applyFlags(cmd *cobra.Command, cfg *config.Config) {
	if p, _ := cmd.Flags().GetString("provider"); p != "" {
		cfg.Provider = p
	}
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		cfg.Model = m
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
		return nil, fmt.Errorf("no API key for %s. Run 'dcode login' to set up authentication.\n\nOr set environment variable:\n  export %s=your-api-key",
			cfg.Provider, getEnvVarName(cfg.Provider))
	}

	prov, err := provider.CreateProvider(cfg.Provider, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", cfg.Provider, err)
	}

	return prov, nil
}

func getEnvVarName(provider string) string {
	envVars := map[string]string{
		"anthropic":  "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"copilot":    "GITHUB_TOKEN",
		"google":     "GOOGLE_API_KEY",
		"groq":       "GROQ_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}
	if v, ok := envVars[provider]; ok {
		return v
	}
	return strings.ToUpper(provider) + "_API_KEY"
}

func formatContextSize(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	return fmt.Sprintf("%dK", tokens/1000)
}
