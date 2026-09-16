package main

import (
	"dun/dun"
	_ "embed"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
)

//go:embed Icon.png
var appIcon []byte

func main() {
	// Any invocation with command-line args is treated as the tiny
	// CLI path (not the GUI) -- e.g. `dunnit DONE "Finished the frob
	// @30m"` appends a ledger entry the same way Daybook's Save
	// button would, without launching the Fyne UI at all. This lets
	// other tools (scripts, LLM-driven workflows, etc) integrate
	// Dunnit entries into a workflow. Deliberately narrow: no flags,
	// no subcommands, exactly CATEGORY + message (2 args) is the only
	// valid shape -- anything else (0 args launches the GUI as
	// normal; 1 or 3+ args is a usage error) is handled below.
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}

	fmt.Println("Starting Dunnit")

	a := dun.MakeUI()
	(*a).SetIcon(fyne.NewStaticResource("Icon.png", appIcon))
	w := dun.BuildMainWindow(*a)
	s := dun.Schedule(*a, w)
	defer s.Shutdown()

	(*a).Run()
}

// runCLI validates args and, if valid, appends the message to today's
// ledger exactly as Daybook's Save button would (dun.RecordActivity),
// then returns a process exit code (0 on success, 1 on bad usage).
// Kept deliberately dumb/tiny per design: no optional flags, just
// "were exactly 2 args given, does the category exist" -- everything
// else (mins parsing, tags, etc) is left to the caller to encode
// directly in message, same as typing into Daybook's entry box.
func runCLI(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: dunnit CATEGORY 'message to record'")
		return 1
	}
	category, message := args[0], args[1]
	if !dun.CategoryExists(category) {
		fmt.Fprintf(os.Stderr, "dunnit: unknown category %q\n", category)
		fmt.Fprintln(os.Stderr, "usage: dunnit CATEGORY 'message to record'")
		return 1
	}
	if err := dun.RecordActivity(message, category); err != nil {
		fmt.Fprintf(os.Stderr, "dunnit: could not record entry: %v\n", err)
		return 1
	}
	return 0
}
