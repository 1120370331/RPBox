package service

import "testing"

func TestStoryArchiveTarget(t *testing.T) {
	tests := []struct {
		name    string
		entries int
		want    int
	}{
		{name: "zero", entries: 0, want: 0},
		{name: "below bucket", entries: 9, want: 0},
		{name: "one bucket", entries: 10, want: 1},
		{name: "multiple buckets", entries: 39, want: 3},
		{name: "cap", entries: 999, want: StoryArchiveDailyMaxExp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := storyArchiveTarget(tt.entries); got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestStoryArchiveLogDay(t *testing.T) {
	if got := storyArchiveLogDay("story_archive_progress", "2026-04-09-archive-5"); got != "2026-04-09" {
		t.Fatalf("expected runtime day to be parsed, got %q", got)
	}

	if got := storyArchiveLogDay(storyArchiveProgressBackfillAction, "2026-04-09:5"); got != "2026-04-09" {
		t.Fatalf("expected backfill day to be parsed, got %q", got)
	}
}
