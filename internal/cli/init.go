package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"golang.org/x/term"

	"wtf/internal/color"
	"wtf/internal/config"
	"wtf/internal/provider"
	"wtf/internal/state"
	"wtf/shell"
)

var initCmd = &cobra.Command{
	Use:   "init [zsh|bash]",
	Short: `Print shell integration code (add to your rc: eval "$(wtf init zsh)")`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := ""
		if len(args) == 1 {
			sh = args[0]
		} else {
			sh = filepath.Base(os.Getenv("SHELL"))
		}
		switch sh {
		case "zsh":
			fmt.Print(shell.ZSH)
		case "bash":
			fmt.Print(shell.BASH)
		default:
			return fmt.Errorf("unsupported shell %q (supported: zsh, bash)", sh)
		}
		if term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Fprintln(os.Stderr, color.Dim.Render(`# add to your rc file:  eval "$(wtf init `+sh+`)"`))
		}
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check installation, hooks, and providers",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	setupColor()
	ok, bad, warn := color.OK.Render("✓"), color.FAIL.Render("✗"), color.WARN.Render("!")

	cfg := config.Load()
	fmt.Printf("%s config: %s\n", ok, config.ConfigPath())
	if b, err := os.ReadFile(config.ConfigPath()); err == nil && len(strings.TrimSpace(string(b))) > 0 {
		fmt.Printf("  %s\n", color.Dim.Render(strings.TrimSpace(string(b))))
	}

	provName := firstNonEmpty(flagProvider, os.Getenv("WTF_PROVIDER"), cfg.Provider)
	if provName == "" {
		provName = provider.Auto()
	}
	if provName == "" {
		fmt.Printf("%s no provider installed\n", bad)
	} else {
		model := firstNonEmpty(flagModel, cfg.ModelFor(provName))
		line := provName
		if model != "" {
			line += " / " + model
		}
		fmt.Printf("%s provider: %s\n", ok, line)
	}

	if l, err := state.Read(); err != nil {
		fmt.Printf("%s hooks: no state recorded — add: eval \"$(wtf init zsh)\"\n", bad)
	} else {
		fmt.Printf("%s hooks: last command %q, exit %d\n", ok, trunc(l.Cmd, 48), l.Exit)
	}

	infos := provider.All()
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	for _, in := range infos {
		if in.Path == "" {
			fmt.Printf("%s %s: not on PATH\n", bad, in.Name)
			continue
		}
		note := in.Note
		if note != "" {
			note = " — " + note
		}
		fmt.Printf("%s %s: %s%s\n", ok, in.Name, in.Path, note)
	}

	if st, err := newestStateTime(); err == nil && time.Since(st) > time.Hour {
		fmt.Printf("%s hooks: last recorded command was %s ago — is the shell with the hooks still open?\n",
			warn, humanAge(time.Since(st)))
	}
	return nil
}

func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func trunc(s string, n int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
