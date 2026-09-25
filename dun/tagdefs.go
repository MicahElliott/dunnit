package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/BurntSushi/toml"
)

// TagDefinition is the optional, current description of a ledger tag. The
// tag name itself is the key in tags.toml and is deliberately not encoded as
// a field, so ledger syntax (#foo) does not leak into the definition format.
type TagDefinition struct {
	Name        string   `toml:"-"`
	Title       string   `toml:"title"`
	Summary     string   `toml:"summary"`
	Description string   `toml:"description"`
	URL         string   `toml:"url"`
	Kind        string   `toml:"kind"`
	Status      string   `toml:"status"`
	Aliases     []string `toml:"aliases"`
	Parent      string   `toml:"parent"`
}

type tagDefinitionsFile struct {
	Tags map[string]TagDefinition `toml:"tags"`
}

func tagDefinitionsPath() string {
	return filepath.Join(DunnitDir(), "tags.toml")
}

// normalizeTagDefinitionName returns a bare tag name, accepting either foo
// or #foo at the API/UI boundary. It uses the same token grammar as ledger
// tags, while rejecting surrounding prose and whitespace.
func normalizeTagDefinitionName(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "#")
	if raw == "" {
		return "", fmt.Errorf("tag name cannot be empty")
	}
	match := tagPattern.FindString("#" + raw)
	if match != "#"+raw {
		return "", fmt.Errorf("invalid tag name %q", raw)
	}
	return raw, nil
}

func normalizeTagDefinitionList(values []string) ([]string, error) {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		name, err := normalizeTagDefinitionName(value)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out, nil
}

func loadTagDefinitions() (map[string]TagDefinition, error) {
	path := tagDefinitionsPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return map[string]TagDefinition{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat tag definitions: %w", err)
	}

	var file tagDefinitionsFile
	if _, err := toml.DecodeFile(path, &file); err != nil {
		return nil, fmt.Errorf("decode tag definitions: %w", err)
	}
	if file.Tags == nil {
		file.Tags = map[string]TagDefinition{}
	}

	definitions := make(map[string]TagDefinition, len(file.Tags))
	for rawName, definition := range file.Tags {
		name, err := normalizeTagDefinitionName(rawName)
		if err != nil {
			return nil, fmt.Errorf("tag definition %q: %w", rawName, err)
		}
		for existing := range definitions {
			if strings.EqualFold(existing, name) {
				return nil, fmt.Errorf("duplicate tag definition %q", name)
			}
		}
		aliases, err := normalizeTagDefinitionList(definition.Aliases)
		if err != nil {
			return nil, fmt.Errorf("tag definition %q aliases: %w", name, err)
		}
		definition.Name = name
		definition.Aliases = aliases
		definitions[name] = definition
	}
	return definitions, nil
}

// LoadTagDefinition looks up a profile by its tag or one of its aliases.
// Definition lookup is case-insensitive because ledger queries already are;
// the original spelling in the ledger remains untouched.
func LoadTagDefinition(tag string) (TagDefinition, bool, error) {
	name, err := normalizeTagDefinitionName(tag)
	if err != nil {
		return TagDefinition{}, false, err
	}
	definitions, err := loadTagDefinitions()
	if err != nil {
		return TagDefinition{}, false, err
	}
	if definition, ok := findTagDefinition(definitions, name); ok {
		return definition, true, nil
	}
	return TagDefinition{}, false, nil
}

func findTagDefinition(definitions map[string]TagDefinition, name string) (TagDefinition, bool) {
	for key, definition := range definitions {
		if strings.EqualFold(key, name) {
			definition.Name = key
			return definition, true
		}
	}
	for key, definition := range definitions {
		for _, alias := range definition.Aliases {
			if strings.EqualFold(alias, name) {
				definition.Name = key
				return definition, true
			}
		}
	}
	return TagDefinition{}, false
}

// SaveTagDefinition writes the current profile while preserving the other
// profiles in tags.toml.
func SaveTagDefinition(definition TagDefinition) error {
	name, err := normalizeTagDefinitionName(definition.Name)
	if err != nil {
		return err
	}
	aliases, err := normalizeTagDefinitionList(definition.Aliases)
	if err != nil {
		return fmt.Errorf("tag definition %q aliases: %w", name, err)
	}
	definition.Name = ""
	definition.Aliases = aliases

	definitions, err := loadTagDefinitions()
	if err != nil {
		return err
	}
	for key := range definitions {
		if strings.EqualFold(key, name) {
			delete(definitions, key)
		}
	}
	definitions[name] = definition

	if err := os.MkdirAll(DunnitDir(), 0755); err != nil {
		return fmt.Errorf("create dunnit dir: %w", err)
	}
	f, err := os.CreateTemp(DunnitDir(), ".tags.toml-*")
	if err != nil {
		return fmt.Errorf("create tag definitions temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	if err := toml.NewEncoder(f).Encode(tagDefinitionsFile{Tags: definitions}); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode tag definitions: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close tag definitions temp file: %w", err)
	}
	if err := os.Rename(tmpPath, tagDefinitionsPath()); err != nil {
		return fmt.Errorf("replace tag definitions: %w", err)
	}
	return nil
}

func tagDefinitionSection(parent fyne.Window, tag string, onChanged func()) fyne.CanvasObject {
	section := container.NewVBox()
	definition, found, err := LoadTagDefinition(tag)
	if err != nil {
		section.Add(widget.NewLabelWithStyle("Tag definition error: "+err.Error(), fyne.TextAlignLeading,
			fyne.TextStyle{Italic: true}))
	} else if found {
		heading := "#" + definition.Name
		if definition.Title != "" {
			heading += " · " + definition.Title
		}
		section.Add(widget.NewLabelWithStyle(heading, fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true}))

		if definition.Summary != "" {
			section.Add(wrappedTagDefinitionLabel(definition.Summary))
		}
		metadata := tagDefinitionMetadata(definition)
		if metadata != "" {
			section.Add(widget.NewLabelWithStyle(metadata, fyne.TextAlignLeading,
				fyne.TextStyle{Italic: true}))
		}
		if definition.URL != "" {
			if target, ok := parseHTTPURL(definition.URL); ok {
				section.Add(widget.NewHyperlink(definition.URL, target))
			} else {
				section.Add(wrappedTagDefinitionLabel("Link: " + definition.URL))
			}
		}
		if strings.TrimSpace(definition.Description) != "" {
			description := newReportRichText(definition.Description)
			description.Wrapping = fyne.TextWrapWord
			section.Add(description)
		}
	} else {
		section.Add(widget.NewLabelWithStyle("#"+strings.TrimPrefix(tag, "#"),
			fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		section.Add(widget.NewLabel("No definition yet. Add one for a summary, description, or useful link."))
	}

	editLabel := "Define tag"
	if found {
		editLabel = "Edit definition"
	}
	section.Add(widget.NewButtonWithIcon(editLabel, theme.Icon(theme.IconNameDocumentCreate), func() {
		showTagDefinitionEditor(parent, tag, onChanged)
	}))
	return section
}

func wrappedTagDefinitionLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func tagDefinitionMetadata(definition TagDefinition) string {
	var parts []string
	if definition.Status != "" {
		parts = append(parts, definition.Status)
	}
	if definition.Kind != "" {
		parts = append(parts, definition.Kind)
	}
	if definition.Parent != "" {
		parts = append(parts, "parent #"+strings.TrimPrefix(definition.Parent, "#"))
	}
	if len(definition.Aliases) > 0 {
		aliases := make([]string, len(definition.Aliases))
		for i, alias := range definition.Aliases {
			aliases[i] = "#" + alias
		}
		parts = append(parts, "aliases "+strings.Join(aliases, ", "))
	}
	return strings.Join(parts, " · ")
}

func showTagDefinitionEditor(parent fyne.Window, tag string, onSave func()) {
	definition, found, err := LoadTagDefinition(tag)
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}
	if !found {
		name, normalizeErr := normalizeTagDefinitionName(tag)
		if normalizeErr != nil {
			dialog.ShowError(normalizeErr, parent)
			return
		}
		definition.Name = name
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetText(definition.Title)
	titleEntry.SetPlaceHolder("Human-readable title (optional)")

	summaryEntry := widget.NewEntry()
	summaryEntry.SetText(definition.Summary)
	summaryEntry.SetPlaceHolder("One-sentence summary (optional)")

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetText(definition.Description)
	descriptionEntry.SetPlaceHolder("Longer Markdown description (optional)")
	descriptionEntry.SetMinRowsVisible(7)

	urlEntry := widget.NewEntry()
	urlEntry.SetText(definition.URL)
	urlEntry.SetPlaceHolder("https://... (optional)")

	kindEntry := widget.NewEntry()
	kindEntry.SetText(definition.Kind)
	kindEntry.SetPlaceHolder("project, ticket, topic... (optional)")

	statusEntry := widget.NewEntry()
	statusEntry.SetText(definition.Status)
	statusEntry.SetPlaceHolder("active, paused, archived... (optional)")

	aliasesEntry := widget.NewEntry()
	aliasesEntry.SetText(strings.Join(addTagPrefixes(definition.Aliases), " "))
	aliasesEntry.SetPlaceHolder("#old-name #another-name (optional)")

	parentEntry := widget.NewEntry()
	parentEntry.SetText(definition.Parent)
	parentEntry.SetPlaceHolder("Parent tag (optional)")

	form := container.NewVBox(
		widget.NewLabel("Tag: #"+definition.Name),
		widget.NewLabel("Title"), titleEntry,
		widget.NewLabel("Summary"), summaryEntry,
		widget.NewLabel("Description"), descriptionEntry,
		widget.NewLabel("Link"), urlEntry,
		widget.NewLabel("Kind"), kindEntry,
		widget.NewLabel("Status"), statusEntry,
		widget.NewLabel("Aliases"), aliasesEntry,
		widget.NewLabel("Parent"), parentEntry,
	)

	d := dialog.NewCustomWithoutButtons("Tag Definition", container.NewVScroll(form), parent)
	save := func() {
		aliases := strings.Fields(aliasesEntry.Text)
		parentName := strings.TrimSpace(parentEntry.Text)
		if parentName != "" {
			name, normalizeErr := normalizeTagDefinitionName(parentName)
			if normalizeErr != nil {
				dialog.ShowError(fmt.Errorf("parent: %w", normalizeErr), parent)
				return
			}
			parentName = name
		}
		updated := TagDefinition{
			Name:        definition.Name,
			Title:       strings.TrimSpace(titleEntry.Text),
			Summary:     strings.TrimSpace(summaryEntry.Text),
			Description: strings.TrimSpace(descriptionEntry.Text),
			URL:         strings.TrimSpace(urlEntry.Text),
			Kind:        strings.TrimSpace(kindEntry.Text),
			Status:      strings.TrimSpace(statusEntry.Text),
			Aliases:     aliases,
			Parent:      parentName,
		}
		if updated.URL != "" {
			if _, ok := parseHTTPURL(updated.URL); !ok {
				dialog.ShowError(fmt.Errorf("link must be a valid http:// or https:// URL"), parent)
				return
			}
		}
		if err := SaveTagDefinition(updated); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		if onSave != nil {
			onSave()
		}
	}

	d.SetButtons([]fyne.CanvasObject{
		widget.NewButton("Cancel", func() { d.Hide() }),
		widget.NewButton("Save", save),
	})
	d.Resize(fyne.NewSize(560, 560))
	d.Show()
}

func addTagPrefixes(names []string) []string {
	withPrefix := make([]string, len(names))
	for i, name := range names {
		withPrefix[i] = "#" + strings.TrimPrefix(name, "#")
	}
	return withPrefix
}
