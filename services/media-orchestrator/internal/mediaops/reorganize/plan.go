package reorganize

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/naming"
)

type Plan struct {
	Source     string
	Target     string
	TargetFile string
	Operation  string
	Action     string
	Reason     string
}

func BuildPlan(root, source, externalID string, parsed naming.ParsedName) Plan {
	if parsed.Title == "" {
		return Plan{Source: source, Operation: "move", Action: "pending_review", Reason: "empty parsed title"}
	}
	group := LetterGroup(parsed.Title)
	name := parsed.Title
	if parsed.Year != nil {
		name = fmt.Sprintf("%s (%d)", name, *parsed.Year)
	}
	if externalID != "" {
		name += " {" + strings.ReplaceAll(externalID, ":", "-") + "}"
	}
	name = sanitize(name)
	category := parsed.Category
	if category == "" {
		category = "pending-review"
	}
	target := filepath.Join(root, category, group, name)
	if parsed.Kind == "series" || parsed.Kind == "anime" || parsed.Kind == "cartoon" {
		if parsed.Season != nil {
			target = filepath.Join(target, fmt.Sprintf("Season %02d", *parsed.Season))
		}
	}
	targetFile := filepath.Join(target, name+filepath.Ext(source))
	if parsed.Season != nil && parsed.Episode != nil {
		targetFile = filepath.Join(target, fmt.Sprintf("%s S%02dE%02d%s", name, *parsed.Season, *parsed.Episode, filepath.Ext(source)))
	}
	action, reason := "pending_review", "identity not yet approved"
	if externalID != "" {
		action, reason = "planned", "identity approved; no file operation executed"
	}
	return Plan{Source: source, Target: target, TargetFile: targetFile, Operation: "move", Action: action, Reason: reason}
}

func LetterGroup(title string) string {
	title = strings.TrimSpace(strings.ToUpper(title))
	if title == "" {
		return "pending-review"
	}
	first := []rune(title)[0]
	if first >= '0' && first <= '9' {
		return "0-9"
	}
	groups := []struct {
		start, end rune
		name       string
	}{{'A', 'C', "A-C"}, {'D', 'F', "D-F"}, {'G', 'I', "G-I"}, {'J', 'L', "J-L"}, {'M', 'O', "M-O"}, {'P', 'R', "P-R"}, {'S', 'U', "S-U"}, {'V', 'X', "V-X"}, {'Y', 'Z', "Y-Z"}}
	for _, group := range groups {
		if first >= group.start && first <= group.end {
			return group.name
		}
	}
	return "pending-review"
}

var invalidWindowsName = regexp.MustCompile(`[<>:"/\\|?*]`)

func sanitize(value string) string {
	value = invalidWindowsName.ReplaceAllString(value, "-")
	value = strings.TrimRight(value, " .")
	if value == "" {
		return "pending-review"
	}
	return value
}

