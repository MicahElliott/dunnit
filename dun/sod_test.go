package dun

import (
	"testing"
	"time"
)

func TestStartOfDayPending(t *testing.T) {
	loc := time.Local
	day := time.Date(2026, time.September, 14, 0, 0, 0, 0, loc)

	tests := []struct {
		name string
		cfg  Config
		now  time.Time
		want bool
	}{
		{
			name: "before kickoff",
			cfg:  Config{KickoffDayEnabled: true, DayStart: "08:00"},
			now:  day.Add(7 * time.Hour),
			want: true,
		},
		{
			name: "kickoff already ran",
			cfg: Config{
				KickoffDayEnabled:  true,
				DayStart:           "08:00",
				LastStartOfDayDate: "2026-09-14",
			},
			now:  day.Add(9 * time.Hour),
			want: false,
		},
		{
			name: "kickoff disabled",
			cfg:  Config{DayStart: "08:00"},
			now:  day.Add(9 * time.Hour),
			want: false,
		},
		{
			name: "weekend",
			cfg:  Config{KickoffDayEnabled: true, DayStart: "08:00"},
			now:  day.AddDate(0, 0, 5).Add(9 * time.Hour),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startOfDayPending(tt.cfg, tt.now); got != tt.want {
				t.Fatalf("startOfDayPending() = %v, want %v", got, tt.want)
			}
		})
	}
}
