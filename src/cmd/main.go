package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"llmclientwrapper/src/internal/domain"
	"llmclientwrapper/src/internal/infrastructure/config"
	"llmclientwrapper/src/internal/infrastructure/llm/router"
	"llmclientwrapper/src/internal/infrastructure/memory"
	"llmclientwrapper/src/internal/infrastructure/prompt"
	infratools "llmclientwrapper/src/internal/infrastructure/tools"

	"github.com/spf13/cobra"
)

// ANSI colour helpers — no external dependency required.
const (
	reset       = "\033[0m"
	bold        = "\033[1m"
	dim         = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
)

func cyan(s string) string      { return colorCyan + s + reset }
func green(s string) string     { return colorGreen + s + reset }
func yellow(s string) string    { return colorYellow + s + reset }
func red(s string) string       { return colorRed + s + reset }
func faint(s string) string     { return dim + s + reset }
func emphasize(s string) string { return bold + s + reset }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		modelFlag      string
		systemFileFlag string
	)

	cmd := &cobra.Command{
		Use:   "llmclientwrapper",
		Short: "Interactive LLM conversation session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), modelFlag, systemFileFlag)
		},
	}

	cmd.Flags().StringVar(&modelFlag, "model", "", "Model alias to use (e.g. sonnet-4.6, devstral)")
	cmd.Flags().StringVar(&systemFileFlag, "system-file", defaultSystemPromptPath(), "Path to a Markdown system prompt file")

	_ = cmd.MarkFlagRequired("model")

	return cmd
}

func run(ctx context.Context, modelAlias, systemFile string) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	r := router.New(cfg)
	client, err := r.Get(domain.Model(modelAlias))
	if err != nil {
		return err
	}

	pp := buildPromptProvider(systemFile)
	tools := infratools.New(cfg).All()

	store := memory.NewStore()
	manager := domain.NewConversationManager(client, modelAlias, store, pp, tools, ConsoleUsageReporter{}, cfg.ToolsMaxConcurrent)
	currentModel := modelAlias

	fmt.Print(cyan(bold+"Session started."+reset) + `
` +
		faint(" Commands:\n") +
		faint("  /model  — switch models\n") +
		faint("  /prompt — show system prompt\n") +
		faint("  /tools  — list available tools\n") +
		faint("  /q      — quit\n"))
	history := NewHistory(historyFilePath())
	lr := NewLineReader(history)
	for {
		fmt.Println()
		input, err := lr.ReadLine(green(bold+"You"+reset+":") + " ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			handleSlashCommand(ctx, input, r, pp, manager, &currentModel, lr, tools)
			continue
		}
		history.Add(input)

		answer, err := manager.Chat(ctx, input)
		if err != nil {
			fmt.Printf("\n%s %s\n", red("Error:"), err.Error())
			continue
		}

		fmt.Printf("\n%s %s\n", cyan(bold+"Assistant"+reset+":"), answer)
	}

	fmt.Println("\n" + faint("Session ended."))
	return nil
}

func buildPromptProvider(systemFile string) domain.PromptProvider {
	return prompt.NewFileProvider(systemFile)
}

func defaultSystemPromptPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "system_prompt.md"
	}
	return filepath.Join(filepath.Dir(exe), "system_prompt.md")
}

func historyFilePath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".llmclientwrapper_history")
	}
	return ".llmclientwrapper_history"
}

func handleSlashCommand(ctx context.Context, input string, r *router.Router, pp domain.PromptProvider, manager *domain.ConversationManager, currentModel *string, lr *LineReader, tools []domain.Tool) {
	cmd := strings.Fields(input)[0]
	switch cmd {
	case "/model":
		cmdModel(r, manager, currentModel, lr)
	case "/prompt":
		cmdPrompt(ctx, pp)
	case "/tools":
		cmdTools(tools)
	case "/q":
		cmdQuit()
	default:
		fmt.Printf("Unknown command %s. Available commands: %s, %s, %s, %s\n",
			red(cmd), yellow("/model"), yellow("/prompt"), yellow("/tools"), yellow("/q"))
	}
}

func cmdTools(tools []domain.Tool) {
	if len(tools) == 0 {
		fmt.Println(faint("(no tools registered)"))
		return
	}
	fmt.Println("\n" + emphasize("Available tools:"))
	for _, t := range tools {
		fmt.Printf("  %s\n    %s\n", cyan(t.Name()), t.Description())
	}
}

func cmdQuit() {
	fmt.Println(faint("Exiting session."))
	os.Exit(0)
}

func cmdPrompt(ctx context.Context, pp domain.PromptProvider) {
	text, err := pp.SystemPrompt(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error loading prompt: ")+err.Error())
		return
	}
	if text == "" {
		fmt.Println(faint("(no system prompt)"))
		return
	}
	fmt.Printf("\n%s\n%s\n%s\n", faint("--- system prompt ---"), text, faint("--- end ---"))
}

func cmdModel(r *router.Router, manager *domain.ConversationManager, currentModel *string, lr *LineReader) {
	models := domain.SupportedModels()
	slices.Sort(models)

	fmt.Println("\n" + emphasize("Available models:"))
	for i, m := range models {
		d, _ := domain.Lookup(m)
		if string(m) == *currentModel {
			fmt.Printf("  [%d] %s %s %s\n", i+1, cyan(fmt.Sprintf("%-14s", m)), faint("("+string(d.Provider)+")"), green("← current"))
		} else {
			fmt.Printf("  [%d] %-14s %s\n", i+1, m, faint("("+string(d.Provider)+")"))
		}
	}

	choice, err := lr.ReadLine(fmt.Sprintf("Choose [1-%d]: ", len(models)))
	if err != nil {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(choice))
	if err != nil || n < 1 || n > len(models) {
		fmt.Println(yellow("Invalid choice, keeping current model."))
		return
	}

	selected := models[n-1]
	client, err := r.Get(selected)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error building client: ")+err.Error())
		return
	}
	manager.SetClient(client, string(selected))
	*currentModel = string(selected)
	fmt.Printf("Switched to %s.\n", green(string(selected)))
}
