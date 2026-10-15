package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"wtf/internal/color"
	"wtf/internal/prompt"
	"wtf/internal/provider"
)

var fixCmd = &cobra.Command{
	Use:          "fix",
	Short:        "Suggest a corrected command, then run it after confirmation",
	SilenceUsage: true,
	RunE:         runFix,
}

func runFix(cmd *cobra.Command, args []string) error {
	setupColor()
	f, err := gather()
	if err != nil {
		return err
	}
	prov, model, err := resolve()
	if err != nil {
		return err
	}
	ctx, cancel := reqContext()
	defer cancel()

	sys, user := prompt.Build(prompt.Fix, f)
	stop := startSpinner("thinking via " + prov.Name())
	out, err := prov.Complete(ctx, provider.Request{System: sys, Prompt: user, Model: model, Cwd: f.Cwd})
	stop()
	if err != nil {
		return err
	}
	fixed, ok := prompt.PostFix(out)
	if !ok {
		color.Warn("no reliable fix found — try: wtf explain")
		return nil
	}

	printHeader(f)
	fmt.Println(color.Cmd.Render("$ " + fixed))

	if !confirm("Run it?") {
		fmt.Println(color.Dim.Render("skipped"))
		return nil
	}
	return runInteractive(fixed, f.Cwd)
}

func confirm(q string) bool {
	fmt.Printf("%s [y/N] ", color.Dim.Render(q))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

func runInteractive(command, cwd string) error {
	c := exec.Command("/bin/sh", "-c", command)
	c.Dir = cwd
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	return nil
}
