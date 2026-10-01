package dun

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestEntryConstructorsDoNotCaptureParentWheel(t *testing.T) {
	single := newSingleLineEntry()
	if single.Wrapping != fyne.TextWrapOff || single.Scroll != fyne.ScrollNone {
		t.Fatalf("single-line entry settings = wrapping %v, scroll %v; want wrap off and scroll none", single.Wrapping, single.Scroll)
	}

	multi := newMultiLineEntry()
	if !multi.MultiLine {
		t.Fatal("multiline constructor returned a single-line entry")
	}
	if multi.Wrapping != fyne.TextWrapOff || multi.Scroll != fyne.ScrollNone {
		t.Fatalf("multiline entry settings = wrapping %v, scroll %v; want wrap off and scroll none", multi.Wrapping, multi.Scroll)
	}
}
