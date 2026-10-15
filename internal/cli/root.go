// Package cli wires the wtf commands.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"wtf/internal/color"
	"wtf/internal/config"
	"wtf/internal/facts"
	"wtf/internal/prompt"
	"wtf/internal/provider"
	"wtf/internal/state"
)

// Version is the wtf release version.
const Version = "0.1.0"

var (
	flagProvider string
	flagModel    string
	flagNoColor  bool
)

var rootCmd = &cobra.Command{
	Use:   "wtf",
	Short: "Explain your last shell command — what happened, why, and how to fix it",
	Long: `wtf inspects the command you just ran and answers in one or two sentences:
what happened, why it failed, and (with "wtf fix") how to correct it.

Commands:
  wtf            did the last command succeed or fail, and why?
  wtf fix        corrected command — printed, then run after confirmation
  wtf explain    what happened and why, in 2-3 sentences
  wtf use        interactive picker for provider and model
  wtf doctor     check installation, hooks, and providers
  wtf init       print shell integration code
  wtf shell      run your shell with full output recording`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalysis(prompt.StatusOK, prompt.StatusFail)
	},
}

// Execute runs the CLI.
func Execute() error {
	rootCmd.PersistentFlags().StringVar(&flagProvider, "provider", "", "LLM provider: claude|codex|opencode|ollama (default: auto)")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "model name (default: from config, then provider default)")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")

	rootCmd.AddCommand(explainCmd, fixCmd, useCmd, initCmd, doctorCmd, shellCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		color.Error("wtf: " + err.Error())
		return err
	}
	return nil
}

func setupColor() {
	if flagNoColor {
		color.Disable()
	}
}

// runAnalysis is the shared body of `wtf` and `wtf explain`.
func runAnalysis(okMode, failMode prompt.Mode) error {
	setupColor()
	f, err := gather()
	if err != nil {
		return err
	}
	mode := failMode
	if f.OK {
		mode = okMode
	}
	prov, model, err := resolve()
	if err != nil {
		return err
	}
	ctx, cancel := reqContext()
	defer cancel()

	sys, user := prompt.Build(mode, f)
	stop := startSpinner("thinking via " + prov.Name())
	out, err := prov.Complete(ctx, provider.Request{System: sys, Prompt: user, Model: model, Cwd: f.Cwd})
	stop()
	if err != nil {
		return err
	}
	printHeader(f)
	fmt.Println(color.Msg.Render(prompt.Post(mode, out)))
	return nil
}

func gather() (facts.Facts, error) {
	l, err := state.Read()
	if errors.Is(err, state.ErrNoState) {
		return facts.Facts{}, fmt.Errorf("no command recorded yet — add the shell hooks: eval \"$(wtf init zsh)\"")
	}
	if err != nil {
		return facts.Facts{}, err
	}
	return facts.Gather(l), nil
}

func resolve() (provider.Provider, string, error) {
	cfg := config.Load()
	name := firstNonEmpty(flagProvider, os.Getenv("WTF_PROVIDER"), cfg.Provider)
	if name == "" {
		name = provider.Auto()
	}
	if name == "" {
		return nil, "", fmt.Errorf("no provider available — install claude, codex, opencode, or ollama")
	}
	p, err := provider.Get(name)
	if err != nil {
		return nil, "", err
	}
	model := firstNonEmpty(flagModel, os.Getenv("WTF_MODEL"), cfg.ModelFor(p.Name()))
	return p, model, nil
}

// reqContext cancels on Ctrl-C and enforces the configured timeout.
func reqContext() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(ctx, config.Load().Timeout())
	return ctx, func() {
		cancel()
		stop()
	}
}

func printHeader(f facts.Facts) {
	var head string
	if f.OK {
		head = color.OK.Render("✓ ok")
	} else {
		head = color.FAIL.Render(fmt.Sprintf("✗ exit %d", f.Exit))
	}
	var meta []string
	if d := f.DurationStr(); d != "" {
		meta = append(meta, d)
	}
	if b := branchOf(f.Git); b != "" {
		meta = append(meta, b)
	}
	if len(meta) > 0 {
		head += " " + color.Dim.Render("· "+strings.Join(meta, " · "))
	}
	fmt.Println(head)
}

// branchOf extracts "main" from "main, clean".
func branchOf(git string) string {
	if git == "" || git == "not a git repo" {
		return ""
	}
	if i := strings.Index(git, ","); i >= 0 {
		return strings.TrimSpace(git[:i])
	}
	return git
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print wtf version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("wtf", Version)
	},
}
