package naming

import "testing"

func TestParseMovieNames(t *testing.T) {
	tests := []struct {
		name, title, category, kind string
		year, resolution            int
	}{
		{"Avatar.Aang.el.ultimo.maestro.del.aire.2026.1080p-Dual-Lat.mkv", "Avatar Aang el ultimo maestro del aire", "01-Movies", "movie", 2026, 1080},
		{"Mortal.Kombat.II.2026.WEB-DL.1080p-Dual-Lat.mkv", "Mortal Kombat II", "01-Movies", "movie", 2026, 1080},
		{"Jane (2023) S01E02 1080p.mkv", "Jane", "02-Series", "series", 2023, 1080},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(`C:\Plex\nueva estructura\`+tt.category+`\`+tt.name, `C:\Plex\nueva estructura`)
			if got.Title != tt.title || got.Category != tt.category || got.Kind != tt.kind || got.Year == nil || *got.Year != tt.year || got.Resolution != tt.resolution {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

