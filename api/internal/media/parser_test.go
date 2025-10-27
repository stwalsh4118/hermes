package media

import (
	"fmt"
	"testing"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantShow    *string
		wantSeason  *int
		wantEpisode *int
		wantTitle   string
	}{
		{
			name:        "standard format with dots",
			input:       "Friends.S01E05.mp4",
			wantShow:    strPtr("Friends"),
			wantSeason:  intPtr(1),
			wantEpisode: intPtr(5),
			wantTitle:   "Friends - S01E05",
		},
		{
			name:        "format with hyphens",
			input:       "Breaking Bad - S02E03 - Episode Name.mkv",
			wantShow:    strPtr("Breaking Bad"),
			wantSeason:  intPtr(2),
			wantEpisode: intPtr(3),
			wantTitle:   "Breaking Bad - S02E03",
		},
		{
			name:        "alternate format (1x07)",
			input:       "Game.of.Thrones.1x07.mp4",
			wantShow:    strPtr("Game of Thrones"),
			wantSeason:  intPtr(1),
			wantEpisode: intPtr(7),
			wantTitle:   "Game of Thrones - S01E07",
		},
		{
			name:        "directory structure",
			input:       "The Office/Season 3/12 - Episode.mp4",
			wantShow:    strPtr("The Office"),
			wantSeason:  intPtr(3),
			wantEpisode: intPtr(12),
			wantTitle:   "The Office - S03E12",
		},
		{
			name:        "directory with Season word",
			input:       "Parks and Recreation/Season 02/05 - Episode Title.mkv",
			wantShow:    strPtr("Parks and Recreation"),
			wantSeason:  intPtr(2),
			wantEpisode: intPtr(5),
			wantTitle:   "Parks and Recreation - S02E05",
		},
		{
			name:        "directory with S## format",
			input:       "The Wire/S04/E03.mp4",
			wantShow:    strPtr("The Wire"),
			wantSeason:  intPtr(4),
			wantEpisode: intPtr(3),
			wantTitle:   "The Wire - S04E03",
		},
		{
			name:        "no pattern match",
			input:       "random_video.mp4",
			wantShow:    nil,
			wantSeason:  nil,
			wantEpisode: nil,
			wantTitle:   "random video",
		},
		{
			name:        "documentary no episode",
			input:       "Documentary.Film.2023.mp4",
			wantShow:    nil,
			wantSeason:  nil,
			wantEpisode: nil,
			wantTitle:   "Documentary Film 2023",
		},
		{
			name:        "lowercase format",
			input:       "sherlock.s01e01.mp4",
			wantShow:    strPtr("sherlock"),
			wantSeason:  intPtr(1),
			wantEpisode: intPtr(1),
			wantTitle:   "sherlock - S01E01",
		},
		{
			name:        "uppercase format",
			input:       "LOST.S03E15.MP4",
			wantShow:    strPtr("LOST"),
			wantSeason:  intPtr(3),
			wantEpisode: intPtr(15),
			wantTitle:   "LOST - S03E15",
		},
		{
			name:        "spaces instead of dots",
			input:       "Doctor Who S05E10.mp4",
			wantShow:    strPtr("Doctor Who"),
			wantSeason:  intPtr(5),
			wantEpisode: intPtr(10),
			wantTitle:   "Doctor Who - S05E10",
		},
		{
			name:        "underscores",
			input:       "The_Mandalorian_S01E08.mp4",
			wantShow:    strPtr("The Mandalorian"),
			wantSeason:  intPtr(1),
			wantEpisode: intPtr(8),
			wantTitle:   "The Mandalorian - S01E08",
		},
		{
			name:        "double digit season and episode",
			input:       "House.S12E23.mp4",
			wantShow:    strPtr("House"),
			wantSeason:  intPtr(12),
			wantEpisode: intPtr(23),
			wantTitle:   "House - S12E23",
		},
		{
			name:        "single digit format",
			input:       "Community.S1E1.mp4",
			wantShow:    strPtr("Community"),
			wantSeason:  intPtr(1),
			wantEpisode: intPtr(1),
			wantTitle:   "Community - S01E01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.input)

			// Check show name
			if !equalStringPtr(result.ShowName, tt.wantShow) {
				t.Errorf("ShowName = %v, want %v", ptrToString(result.ShowName), ptrToString(tt.wantShow))
			}

			// Check season
			if !equalIntPtr(result.Season, tt.wantSeason) {
				t.Errorf("Season = %v, want %v", ptrToInt(result.Season), ptrToInt(tt.wantSeason))
			}

			// Check episode
			if !equalIntPtr(result.Episode, tt.wantEpisode) {
				t.Errorf("Episode = %v, want %v", ptrToInt(result.Episode), ptrToInt(tt.wantEpisode))
			}

			// Check title
			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}

			// Check raw filename
			if result.RawFilename != tt.input {
				t.Errorf("RawFilename = %q, want %q", result.RawFilename, tt.input)
			}
		})
	}
}

func TestCleanShowName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "dots to spaces",
			input: "Show.Name.Here",
			want:  "Show Name Here",
		},
		{
			name:  "underscores to spaces",
			input: "Show_Name_Here",
			want:  "Show Name Here",
		},
		{
			name:  "mixed separators",
			input: "Show.Name_Here",
			want:  "Show Name Here",
		},
		{
			name:  "leading/trailing spaces",
			input: "  Show Name  ",
			want:  "Show Name",
		},
		{
			name:  "multiple spaces",
			input: "Show    Name    Here",
			want:  "Show Name Here",
		},
		{
			name:  "already clean",
			input: "Show Name",
			want:  "Show Name",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanShowName(tt.input)
			if got != tt.want {
				t.Errorf("cleanShowName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSeasonEpisode(t *testing.T) {
	tests := []struct {
		season  int
		episode int
		want    string
	}{
		{season: 1, episode: 1, want: "S01E01"},
		{season: 1, episode: 5, want: "S01E05"},
		{season: 10, episode: 15, want: "S10E15"},
		{season: 2, episode: 23, want: "S02E23"},
		{season: 99, episode: 99, want: "S99E99"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSeasonEpisode(tt.season, tt.episode)
			if got != tt.want {
				t.Errorf("formatSeasonEpisode(%d, %d) = %q, want %q", tt.season, tt.episode, got, tt.want)
			}
		})
	}
}

func TestFormatEpisodeTitle(t *testing.T) {
	tests := []struct {
		show    string
		season  int
		episode int
		want    string
	}{
		{
			show:    "Friends",
			season:  1,
			episode: 5,
			want:    "Friends - S01E05",
		},
		{
			show:    "Breaking Bad",
			season:  5,
			episode: 14,
			want:    "Breaking Bad - S05E14",
		},
		{
			show:    "  Spaced Show  ",
			season:  2,
			episode: 3,
			want:    "Spaced Show - S02E03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatEpisodeTitle(tt.show, tt.season, tt.episode)
			if got != tt.want {
				t.Errorf("formatEpisodeTitle(%q, %d, %d) = %q, want %q", tt.show, tt.season, tt.episode, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  *int
	}{
		{input: "1", want: intPtr(1)},
		{input: "42", want: intPtr(42)},
		{input: "0", want: intPtr(0)},
		{input: "invalid", want: nil},
		{input: "", want: nil},
		{input: "3.14", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt(tt.input)
			if !equalIntPtr(got, tt.want) {
				t.Errorf("parseInt(%q) = %v, want %v", tt.input, ptrToInt(got), ptrToInt(tt.want))
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "just extension", input: ".mp4"},
		{name: "no extension", input: "filename"},
		{name: "multiple extensions", input: "file.name.mp4"},
		{name: "special characters", input: "Show (2023) - S01E01.mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.input)
			// Should not panic
			if result.RawFilename != tt.input {
				t.Errorf("RawFilename = %q, want %q", result.RawFilename, tt.input)
			}
			// Title should always be set
			if result.Title == "" && tt.input != "" && tt.input != ".mp4" {
				t.Error("Title should not be empty for non-empty input")
			}
		})
	}
}

// Helper functions for testing

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrToString(p *string) string {
	if p == nil {
		return "nil"
	}
	return *p
}

func ptrToInt(p *int) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}
