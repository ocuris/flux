package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ocuris/flux"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("flux", flag.ContinueOnError)
	reload := fs.Bool("reload", false, "Enable hot reloading")
	rShort := fs.Bool("r", false, "Enable hot reloading (shorthand)")
	version := fs.Bool("version", false, "Print version")
	vShort := fs.Bool("v", false, "Print version (shorthand)")

	fs.Usage = printHelp
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *version || *vShort {
		fmt.Printf("Flux CLI v%s\n", flux.Version)
		return nil
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printHelp()
		return nil
	}

	commandOrFile := remaining[0]
	appArgs := remaining

	switch commandOrFile {
	case "run":
		if len(remaining) < 2 {
			return fmt.Errorf("run requires a file or command to execute")
		}
		appArgs = remaining[1:]
	case "help":
		printHelp()
		return nil
	case "version":
		fmt.Printf("Flux CLI v%s\n", flux.Version)
		return nil
	}

	shouldReload := *reload || *rShort

	var finalArgs []string
	if strings.HasSuffix(appArgs[0], ".go") {
		if _, err := os.Stat(appArgs[0]); os.IsNotExist(err) {
			return fmt.Errorf("file %s not found", appArgs[0])
		}
		finalArgs = append([]string{"go", "run"}, appArgs...)
	} else {
		finalArgs = appArgs
	}

	if shouldReload {
		reloader := flux.NewReloader(finalArgs...)
		return reloader.Run()
	}

	cmd := exec.Command(finalArgs[0], finalArgs[1:]...)
	cmd.Env = append(os.Environ(), "FLUX_MANAGED=true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func printHelp() {
	fmt.Println("Flux CLI - The high-performance companion for your Go services")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  flux [flags] <target> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -r, --reload    Enable hot-reloading (watches for .go file changes)")
	fmt.Println("  -v, --version   Display current version information")
	fmt.Println("  -h, --help      Display this diagnostic help menu")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  target          The Go file (e.g., main.go) or compiled binary to run")
	fmt.Println("  args...         Optional arguments passed directly to your application")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ▸ Run an application directly:")
	fmt.Println("    flux main.go")
	fmt.Println()
	fmt.Println("  ▸ Run with hot-reloading enabled:")
	fmt.Println("    flux --reload explorer.go")
	fmt.Println()
	fmt.Println("  ▸ Pass arguments to your app:")
	fmt.Println("    flux main.go --port 9000 --debug")
	fmt.Println()
	fmt.Println("For full documentation and support, visit https://github.com/ocuris/flux")
}
