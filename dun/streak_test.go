package dun

import (
	"strings"
	"testing"
	"time"
)

func TestIsTicketTag(t *testing.T) {
	cases := []struct {
		tag  string
		want bool
	}{
		{"#12345", true},
		{"#SCRUM-12345", true},
		{"#scrum-12345", true},
		{"#SCRUM-12345-extra", false},
		{"#planning", false},
		{"#123abc", false},
	}
	for _, tc := range cases {
		if got := isTicketTag(tc.tag); got != tc.want {
			t.Errorf("isTicketTag(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestStreakCalloutCandidates(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.Local)
	var entries []LedgerEntry
	add := func(daysAgo int, category, text string, mins int) {
		date := dateOnly(now.AddDate(0, 0, -daysAgo))
		entries = append(entries, LedgerEntry{
			Date:     date,
			Category: category,
			Text:     text,
			Tags:     extractTags(text),
			People:   extractPeople(text),
			Mins:     mins,
		})
	}

	for daysAgo := 0; daysAgo < 3; daysAgo++ {
		for done := 0; done < 6; done++ {
			add(daysAgo, "DONE", "finished work #alpha", 0)
		}
		add(daysAgo, "DOING", "continue work #alpha ~120m", 120)
		add(daysAgo, "TIL", "learned something", 0)
		add(daysAgo, "WIN", "shipped something", 0)
	}
	add(0, "KUDOS", "recognized a teammate", 0)
	for daysAgo, ticket := range []string{"#SCRUM-100", "#SCRUM-101", "#SCRUM-102", "#12345", "#12346"} {
		add(daysAgo, "DONE", "covered "+ticket, 0)
	}
	for daysAgo, project := range []string{"#beta", "#gamma", "#delta"} {
		add(daysAgo, "DONE", "advanced "+project, 0)
	}
	for daysAgo, person := range []string{"@Alice", "@Bob", "@Carol", "@Dave", "@Eve", "@Frank", "@Grace"} {
		add(daysAgo%5, "MEETING", "worked with "+person, 0)
	}
	for _, want := range []string{
		"🔥 4 consecutive workdays logged",
		"🏁 3 workdays in a row with 6+ DONEs",
		"✨ 5 Hilites this week",
		"🏆 WINs on 3 days this week",
		"🌱 Learned something on 3 days this week",
		"⏱️ Logged 6h of explicit time this week",
		"🌈 Used 3 different Hilite types this week",
		"🤝 Worked with 7 people this week",
		"🎫 Covered 5 tickets in the last 5 workdays",
		"🗂️ Touched 4 projects or topics this week",
		"🔁 Used #alpha in 5 entries this week",
		"🎯 Kept #alpha moving for 3 workdays",
	} {
		found := false
		for _, got := range streakCalloutCandidates(entries, now, 4) {
			if strings.Contains(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("streak callouts did not include %q; got %v", want,
				streakCalloutCandidates(entries, now, 4))
		}
	}
}

func TestSelectStreakCalloutsCapsTheCompactList(t *testing.T) {
	candidates := []string{"one", "two", "three", "four"}
	selected := selectStreakCallouts(append([]string{}, candidates...))
	if len(selected) != maxStreakCallouts {
		t.Fatalf("selected %v, want %d callouts", selected, maxStreakCallouts)
	}
}

func TestStreakCalloutCandidatesExcludeOrdinaryLifecycleFlow(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.Local)
	entries := []LedgerEntry{{Date: now, Category: "TODO"}, {Date: now, Category: "DOING"}, {Date: now, Category: "DONE"}}
	for _, callout := range streakCalloutCandidates(entries, now, 0) {
		if strings.Contains(callout, "TODO") || strings.Contains(callout, "DOING") {
			t.Errorf("ordinary lifecycle flow produced callout %q", callout)
		}
	}
}

func TestStreakCalloutCandidatesExcludeConfiguredTags(t *testing.T) {
	withTempDunnitDir(t)
	cfg := LoadConfig()
	cfg.ReportExcludeTags = []string{"#home"}
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.Local)
	entries := []LedgerEntry{
		{Date: now, Category: "DONE", Text: "finished #home", Tags: []string{"#home"}},
	}
	if got := streakCalloutCandidates(entries, now, 0); len(got) != 0 {
		t.Fatalf("excluded-only entries produced streak callouts: %v", got)
	}
}

func TestCurrentStreakIgnoresExcludedOnlyDays(t *testing.T) {
	withTempDunnitDir(t)
	cfg := LoadConfig()
	cfg.ReportExcludeTags = []string{"#home"}
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	yesterday := previousCarryWorkday(time.Now())
	priorWorkday := previousCarryWorkday(yesterday)
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] DONE cleaned the garage #home",
	})
	writeLedgerLinesForDate(t, priorWorkday, []string{
		"[09:00:00] DONE shipped the fix #work",
	})
	InvalidateLedgerCaches()

	if got := CurrentStreak(); got != 0 {
		t.Fatalf("CurrentStreak() = %d, want 0 after an excluded-only day", got)
	}
}
