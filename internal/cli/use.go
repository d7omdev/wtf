package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"wtf/internal/color"
	"wtf/internal/config"
	"wtf/internal/provider"
)

var useCmd = &cobra.Command{
	Use:          "use [provider] [model]",
	Short:        "Pick the provider and model (interactive picker without args)",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
	RunE:         runUse,
}

func runUse(cmd *cobra.Command, args []string) error {
	setupColor()
	cfg := config.Load()
	switch len(args) {
	case 2:
		return saveSelection(cfg, args[0], args[1])
	case 1:
		if !isInteractive() {
			return fmt.Errorf("the model picker needs a TTY; use: wtf use %s <model>", args[0])
		}
		model, err := pickModel(args[0], cfg.ModelFor(args[0]))
		if err != nil {
			return err
		}
		return saveSelection(cfg, args[0], model)
	default:
		if !isInteractive() {
			return fmt.Errorf("the picker needs a TTY; use: wtf use <provider> <model>")
		}
		prov, err := pickProvider(cfg)
		if err != nil {
			return err
		}
		model, err := pickModel(prov, cfg.ModelFor(prov))
		if err != nil {
			return err
		}
		return saveSelection(cfg, prov, model)
	}
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func pickProvider(cfg config.Config) (string, error) {
	infos := provider.All()
	// configured provider first, then installed ones, then the rest
	sort.SliceStable(infos, func(i, j int) bool {
		if infos[i].Name == cfg.Provider {
			return true
		}
		if infos[j].Name == cfg.Provider {
			return false
		}
		ii, jj := infos[i].Path != "", infos[j].Path != ""
		return ii && !jj
	})

	opts := make([]huh.Option[string], 0, len(infos))
	for _, in := range infos {
		label := in.Name
		if in.Path == "" {
			label += " — not installed"
		} else if in.Note != "" {
			label += " — " + in.Note
		}
		opts = append(opts, huh.NewOption(label, in.Name))
	}

	choice := cfg.Provider
	if choice == "" {
		choice = infos[0].Name
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Provider").
				Description("Where wtf asks for explanations.").
				Options(opts...).
				Value(&choice),
		),
	).Run()
	return choice, err
}

func pickModel(providerName, current string) (string, error) {
	p, err := provider.Get(providerName)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	models, _ := p.Models(ctx) // discovery failure is fine: custom input still works

	const custom = "custom…"
	var opts []huh.Option[string]
	seen := map[string]bool{}
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		opts = append(opts, huh.NewOption(v, v))
	}
	add(current)
	for _, m := range models {
		add(m)
	}
	opts = append(opts, huh.NewOption(custom, custom))

	choice := current
	if choice == "" && len(models) > 0 {
		choice = models[0]
	} else if choice == "" {
		choice = custom
	}

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(providerName + " model").
				Options(opts...).
				Value(&choice),
		),
	).Run(); err != nil {
		return "", err
	}
	if choice != custom {
		return choice, nil
	}

	var typed string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Model name").
				Description("provider-specific, e.g. anthropic/claude-sonnet-5 or qwen2.5-coder:7b").
				Value(&typed),
		),
	).Run(); err != nil {
		return "", err
	}
	typed = strings.TrimSpace(typed)
	if typed == "" {
		return "", fmt.Errorf("no model given")
	}
	return typed, nil
}

func saveSelection(cfg config.Config, prov, model string) error {
	if _, err := provider.Get(prov); err != nil {
		return err
	}
	if cfg.Models == nil {
		cfg.Models = map[string]string{}
	}
	cfg.Provider = prov
	cfg.Models[prov] = model
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Printf("%s using %s / %s  (%s)\n",
		color.OK.Render("✓"), color.Cmd.Render(prov), color.Cmd.Render(model), config.ConfigPath())
	return nil
}
