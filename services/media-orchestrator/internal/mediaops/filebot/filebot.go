package filebot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Command struct {
	Executable string
	Args       []string
}
type Executor interface {
	Run(context.Context, Command) ([]byte, error)
}

type Metadata struct {
	Provider      string   `json:"provider"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Year          *int     `json:"year,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Director      string   `json:"director,omitempty"`
	Actors        []string `json:"actors,omitempty"`
	Rating        float64  `json:"rating,omitempty"`
	Runtime       int      `json:"runtime,omitempty"`
	Country       string   `json:"country,omitempty"`
	Language      string   `json:"language,omitempty"`
	Certification string   `json:"certification,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Edition       string   `json:"edition,omitempty"`
}

const metadataFormat = `{id}|{n}|{y}|{genres}|{director}|{actors}|{rating}|{runtime}|{country}|{language}|{certification}|{tags}|{edition}`

func BuildMetadataCommand(executable, database, query string) Command {
	if executable == "" {
		executable = Executable()
	}
	return Command{Executable: executable, Args: withLanguage([]string{"-list", "--db", database, "--q", query, "--format", metadataFormat, "-non-strict"})}
}

// withLanguage keeps FileBot metadata and rename results aligned with the
// Spanish-first library policy. Set PRA_FILEBOT_LANGUAGE to override it.
func withLanguage(args []string) []string {
	language := strings.TrimSpace(os.Getenv("PRA_FILEBOT_LANGUAGE"))
	if language == "" {
		language = "es"
	}
	return append(args, "--lang", language)
}

func NormalizeQuery(value string) string {
	value = strings.NewReplacer(".", " ", "_", " ").Replace(value)
	value = regexp.MustCompile(`(?i)\b(2160p|1080p|720p|web[- ]?dl|bluray|dual|lat|x264|x265|h264|h265)\b`).ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}
func QueryMetadata(ctx context.Context, executor Executor, database, query string) ([]Metadata, error) {
	if executor == nil {
		return nil, fmt.Errorf("filebot executor is required")
	}
	out, err := executor.Run(ctx, BuildMetadataCommand("", database, query))
	if err != nil {
		return nil, err
	}
	return parseMetadata(string(out), database), nil
}
func parseMetadata(output, provider string) []Metadata {
	result := make([]Metadata, 0)
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "Missing data:") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 13 {
			continue
		}
		m := Metadata{Provider: provider, ID: strings.TrimSpace(parts[0]), Title: strings.TrimSpace(parts[1]), Genres: parseList(parts[3]), Director: strings.TrimSpace(parts[4]), Actors: parseList(parts[5]), Country: strings.TrimSpace(parts[8]), Language: strings.TrimSpace(parts[9]), Certification: strings.TrimSpace(parts[10]), Tags: parseList(parts[11]), Edition: strings.TrimSpace(parts[12])}
		if y, e := strconv.Atoi(strings.TrimSpace(parts[2])); e == nil && y > 0 {
			m.Year = &y
		}
		if rating, e := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64); e == nil && rating >= 0 && rating <= 10 {
			m.Rating = rating
		}
		if runtime, e := strconv.Atoi(strings.TrimSpace(parts[7])); e == nil {
			m.Runtime = runtime
		}
		result = append(result, m)
	}
	return result
}
func parseList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, command Command) ([]byte, error) {
	return exec.CommandContext(ctx, command.Executable, command.Args...).CombinedOutput()
}

func Executable() string {
	if value := os.Getenv("PRA_FILEBOT_PATH"); value != "" {
		return value
	}
	if value := os.Getenv("ProgramFiles"); value != "" {
		return filepath.Join(value, "FileBot", "filebot.exe")
	}
	return "filebot"
}

func BuildRenameCommand(executable, source, database, format string, dryRun bool) Command {
	if executable == "" {
		executable = Executable()
	}
	args := withLanguage([]string{"-rename", source, "--db", database, "--format", format, "--conflict", "skip"})
	if dryRun {
		args = append(args, "--action", "test")
	} else {
		args = append(args, "--action", "move")
	}
	return Command{Executable: executable, Args: args}
}

func BuildRenameCommandInOutput(executable, source, outputRoot, database, format string, dryRun bool) Command {
	command := BuildRenameCommand(executable, source, database, format, dryRun)
	args := make([]string, 0, len(command.Args)+2)
	for i, arg := range command.Args {
		args = append(args, arg)
		if i == 0 {
			args = append(args, "--output", outputRoot)
		}
	}
	command.Args = args
	return command
}

func Rename(ctx context.Context, executor Executor, source, database, format string, dryRun bool) ([]byte, error) {
	if executor == nil {
		return nil, fmt.Errorf("filebot executor is required")
	}
	if !dryRun && strings.TrimSpace(os.Getenv("PRA_ALLOW_FILEBOT_MOVE")) != "1" {
		return nil, fmt.Errorf("FileBot move bloqueado: requiere PRA_ALLOW_FILEBOT_MOVE=1")
	}
	output, err := executor.Run(ctx, BuildRenameCommand("", source, database, format, dryRun))
	if dryRun && err != nil && strings.Contains(string(output), "[TEST]") {
		return output, nil
	}
	return output, err
}

func RenameInOutput(ctx context.Context, executor Executor, source, outputRoot, database, format string, dryRun bool) ([]byte, error) {
	if executor == nil {
		return nil, fmt.Errorf("filebot executor is required")
	}
	if !dryRun && strings.TrimSpace(os.Getenv("PRA_ALLOW_FILEBOT_MOVE")) != "1" {
		return nil, fmt.Errorf("FileBot move bloqueado: requiere PRA_ALLOW_FILEBOT_MOVE=1")
	}
	output, err := executor.Run(ctx, BuildRenameCommandInOutput("", source, outputRoot, database, format, dryRun))
	if dryRun && err != nil && strings.Contains(string(output), "[TEST]") {
		return output, nil
	}
	return output, err
}

type PlexMatch struct {
	Title   string
	Year    *int
	TMDBID  string
	TVDBID  string
	IMDbID  string
	Season  *int
	Episode *int
	AirDate string
}

// SeasonPlexMatch describes the identity and ordering metadata for a season.
// It is used for podcasts, festivals and other custom series where the year
// itself is the season number.
type SeasonPlexMatch struct {
	Title  string
	Year   *int
	TMDBID string
	TVDBID string
	IMDbID string
	Season int
}

type EpisodeName struct {
	Title     string
	AirDate   string
	Artist    string
	Season    int
	Episode   int
	Part      int
	Extension string
}

// FormatEpisodeName keeps date and artist visible for custom series while
// retaining SxxEyy fallback ordering for Plex/Jellyfin-compatible libraries.
func FormatEpisodeName(e EpisodeName) (string, error) {
	if strings.TrimSpace(e.Title) == "" && strings.TrimSpace(e.Artist) == "" {
		return "", fmt.Errorf("episode title or artist is required")
	}
	parts := make([]string, 0, 4)
	if strings.TrimSpace(e.AirDate) != "" {
		parts = append(parts, strings.TrimSpace(e.AirDate))
	} else if e.Season >= 0 && e.Episode > 0 {
		parts = append(parts, fmt.Sprintf("S%02dE%02d", e.Season, e.Episode))
	}
	if strings.TrimSpace(e.Artist) != "" {
		parts = append(parts, strings.TrimSpace(e.Artist))
	}
	if strings.TrimSpace(e.Title) != "" {
		parts = append(parts, strings.TrimSpace(e.Title))
	}
	if e.Part > 0 {
		parts = append(parts, fmt.Sprintf("Part %02d", e.Part))
	}
	ext := strings.TrimSpace(e.Extension)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return strings.Join(parts, " - ") + ext, nil
}

func GenerateSeasonPlexMatch(m SeasonPlexMatch) ([]byte, error) {
	if strings.TrimSpace(m.Title) == "" {
		return nil, fmt.Errorf("plexmatch season title is required")
	}
	if m.Season < 0 {
		return nil, fmt.Errorf("plexmatch season cannot be negative")
	}
	return GeneratePlexMatch(PlexMatch{Title: m.Title, Year: m.Year, TMDBID: m.TMDBID, TVDBID: m.TVDBID, IMDbID: m.IMDbID, Season: &m.Season})
}

func WriteSeasonPlexMatch(dir string, m SeasonPlexMatch) error {
	data, err := GenerateSeasonPlexMatch(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".plexmatch"), data, 0600)
}

func GeneratePlexMatch(m PlexMatch) ([]byte, error) {
	if strings.TrimSpace(m.Title) == "" {
		return nil, fmt.Errorf("plexmatch title is required")
	}
	lines := []string{"Title: " + strings.TrimSpace(m.Title)}
	if m.Year != nil {
		lines = append(lines, "Year: "+strconv.Itoa(*m.Year))
	}
	if m.TMDBID != "" {
		lines = append(lines, "TMDBID: "+m.TMDBID)
	}
	if m.TVDBID != "" {
		lines = append(lines, "TVDBID: "+m.TVDBID)
	}
	if m.IMDbID != "" {
		lines = append(lines, "IMDbID: "+m.IMDbID)
	}
	if m.Season != nil {
		lines = append(lines, "Season: "+strconv.Itoa(*m.Season))
	}
	if m.Episode != nil {
		lines = append(lines, "Episode: "+strconv.Itoa(*m.Episode))
	}
	if m.AirDate != "" {
		lines = append(lines, "AirDate: "+m.AirDate)
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}
func WritePlexMatch(path string, match PlexMatch) error {
	data, err := GeneratePlexMatch(match)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

