package dun

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestTooltipPopupForwardsOwnerClicks(t *testing.T) {
	tests := []struct {
		name      string
		click     fyne.Position
		wantOwner bool
	}{
		{name: "inside owner", click: fyne.NewPos(20, 30), wantOwner: true},
		{name: "outside owner", click: fyne.NewPos(50, 60)},
		{name: "on owner edge", click: fyne.NewPos(40, 50), wantOwner: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerCalls := 0
			eventCalls := 0
			popup := &tooltipPopup{
				ownerPos:  fyne.NewPos(20, 30),
				ownerSize: fyne.NewSize(20, 20),
				ownerTapped: func() {
					ownerCalls++
				},
				ownerTappedEvent: func(*fyne.PointEvent) {
					eventCalls++
				},
			}

			popup.Tapped(&fyne.PointEvent{AbsolutePosition: tt.click})

			if got := ownerCalls; got != 0 {
				t.Fatalf("owner callback called %d times, want 0 when event callback is set", got)
			}
			wantEvents := 0
			if tt.wantOwner {
				wantEvents = 1
			}
			if eventCalls != wantEvents {
				t.Fatalf("event callback called %d times, want %d", eventCalls, wantEvents)
			}
		})
	}
}

func TestTooltipPopupUsesOwnerCallbackForButtons(t *testing.T) {
	called := false
	popup := &tooltipPopup{
		ownerPos:    fyne.NewPos(10, 10),
		ownerSize:   fyne.NewSize(20, 20),
		ownerTapped: func() { called = true },
	}

	popup.Tapped(&fyne.PointEvent{AbsolutePosition: fyne.NewPos(15, 15)})

	if !called {
		t.Fatal("owner callback was not called for a button click")
	}
}
