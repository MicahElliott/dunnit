package main

import (
	"dun/dun"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
)

//go:embed Icon.png
var appIcon []byte

func main() {
	// Any invocation with command-line args is treated as the tiny
	// CLI path (not the GUI) -- e.g. `dunnit DONE "Finished the frob
	// ~30m"` appends a ledger entry the same way Daybook's Save
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
	// Set the app icon before building the tray and scheduler. Fyne uses the
	// app resource for the tray, packaged app, and native OS notifications.
	(*a).SetIcon(fyne.NewStaticResource("Icon.png", appIcon))
	w := dun.BuildMainWindow(*a)
	s := dun.Schedule(*a, w)
	defer s.Shutdown()

	(*a).Run()
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ", ") }

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// runCLI supports the original two-argument entry form plus the tag
// subcommand used by scripts and agent harnesses to maintain tag profiles.
func runCLI(args []string) int {
	if len(args) > 0 && args[0] == "tag" {
		return runTagCLI(args[1:])
	}
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

func runTagCLI(args []string) int {
	fs := flag.NewFlagSet("dunnit tag", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		title       string
		summary     string
		description string
		url         string
		kind        string
		status      string
		parent      string
		aliases     stringList
	)
	fs.StringVar(&title, "title", "", "human-readable title")
	fs.StringVar(&summary, "summary", "", "one-sentence summary")
	fs.StringVar(&description, "description", "", "longer Markdown description")
	fs.StringVar(&url, "url", "", "http:// or https:// link")
	fs.StringVar(&kind, "kind", "", "project, ticket, topic, person, team, service, area, or goal")
	fs.StringVar(&status, "status", "", "active, planned, blocked, paused, done, or archived")
	fs.Var(&aliases, "alias", "repeatable alias")
	fs.StringVar(&parent, "parent", "", "parent tag")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: dunnit tag TAG [options]")
		fs.PrintDefaults()
	}

	tagName := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tagName = args[0]
		flagArgs = args[1:]
	}
	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}
	if tagName == "" {
		remaining := fs.Args()
		if len(remaining) != 1 {
			fs.Usage()
			return 1
		}
		tagName = remaining[0]
	} else if len(fs.Args()) > 0 {
		fmt.Fprintln(os.Stderr, "dunnit tag: unexpected positional argument")
		fs.Usage()
		return 1
	}

	definition, found, err := dun.LoadTagDefinition(tagName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dunnit tag: could not load profile: %v\n", err)
		return 1
	}
	if !found {
		definition.Name = tagName
	}
	if fsWasSet(fs, "title") {
		definition.Title = strings.TrimSpace(title)
	}
	if fsWasSet(fs, "summary") {
		definition.Summary = strings.TrimSpace(summary)
	}
	if fsWasSet(fs, "description") {
		definition.Description = strings.TrimSpace(description)
	}
	if fsWasSet(fs, "url") {
		definition.URL = strings.TrimSpace(url)
	}
	if fsWasSet(fs, "kind") {
		definition.Kind = strings.TrimSpace(kind)
	}
	if fsWasSet(fs, "status") {
		definition.Status = strings.TrimSpace(status)
	}
	if fsWasSet(fs, "parent") {
		definition.Parent = strings.TrimSpace(parent)
	}
	if fsWasSet(fs, "alias") {
		var values []string
		for _, value := range aliases {
			values = append(values, strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })...)
		}
		definition.Aliases = values
	}
	if err := dun.SaveTagDefinition(definition); err != nil {
		fmt.Fprintf(os.Stderr, "dunnit tag: could not save profile: %v\n", err)
		return 1
	}
	return 0
}

func fsWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			set = true
		}
	})
	return set
}
