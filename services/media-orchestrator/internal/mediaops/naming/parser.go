package naming

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ParsedName struct {
	Title          string
	Year           *int
	Season         *int
	Episode        *int
	Edition        string
	Resolution     int
	Codec          string
	Source         string
	Audio          string
	Category       string
	Kind           string
	IdentityStatus string
}

var yearPattern = regexp.MustCompile(`(?:^|[ ._()\-])((?:18|19|20|21)\d{2})(?:$|[ ._()\-])`)
var episodePattern = regexp.MustCompile(`(?i)S(\d{1,2})(?:E(\d{1,3}))?`)
var resolutionPattern = regexp.MustCompile(`(?i)(2160|1080|720|480)p`)

func Parse(path, root string) ParsedName {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parsed := ParsedName{Audio: "unknown", IdentityStatus: "unidentified"}
	parsed.Category, parsed.Kind = categoryFromPath(path, root)

	if match := yearPattern.FindStringSubmatch(base); len(match) == 2 {
		if year, err := strconv.Atoi(match[1]); err == nil {
			parsed.Year = &year
			base = strings.Replace(base, match[1], "", 1)
		}
	}
	if match := episodePattern.FindStringSubmatch(base); len(match) > 0 {
		season, _ := strconv.Atoi(match[1])
		parsed.Season = &season
		if match[2] != "" {
			episode, _ := strconv.Atoi(match[2])
			parsed.Episode = &episode
		}
		base = episodePattern.ReplaceAllString(base, " ")
	}
	if match := resolutionPattern.FindStringSubmatch(base); len(match) == 2 {
		parsed.Resolution, _ = strconv.Atoi(match[1])
		base = resolutionPattern.ReplaceAllString(base, " ")
	}
	for _, token := range []struct{ pattern, value string }{
		{`(?i)(web-dl|webdl|bluray|blu-ray|bdrip|hdrip|hdtv|remux)`, "source"},
		{`(?i)(x265|h265|hevc|x264|h264|av1)`, "codec"},
		{`(?i)(dual-lat|dual|latino|castellano|español|spanish|multi)`, "audio"},
		{`(?i)(director.?s cut|extended cut|theatrical cut|uncut|remastered)`, "edition"},
	} {
		r := regexp.MustCompile(token.pattern)
		if match := r.FindString(base); match != "" {
			switch token.value {
			case "source":
				parsed.Source = strings.ToLower(match)
			case "codec":
				parsed.Codec = strings.ToLower(match)
			case "audio":
				parsed.Audio = strings.ToLower(match)
			case "edition":
				parsed.Edition = strings.ToLower(strings.TrimSpace(match))
			}
			base = r.ReplaceAllString(base, " ")
		}
	}
	parsed.Title = cleanTitle(base)
	return parsed
}

func categoryFromPath(path, root string) (string, string) {
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '\\' || r == '/' })
	category := "pending-review"
	for _, part := range parts {
		if strings.HasPrefix(part, "01-") || strings.HasPrefix(part, "02-") || strings.HasPrefix(part, "03-") || strings.HasPrefix(part, "04-") || strings.HasPrefix(part, "05-") || strings.HasPrefix(part, "06-") || strings.HasPrefix(part, "07-") || strings.HasPrefix(part, "08-") || strings.HasPrefix(part, "09-") || strings.HasPrefix(part, "10-") || strings.HasPrefix(part, "11-") {
			category = part
			break
		}
	}
	kind := map[string]string{"01-Movies": "movie", "02-Series": "series", "03-Cartoons": "cartoon", "04-Anime": "anime", "05-Documentaries": "documentary", "06-Novelas": "novel", "07-Comedy": "comedy", "08-Sports": "sports", "09-Music": "music", "10-Podcasts": "podcast", "11-Festivales": "festival"}[category]
	if kind == "" {
		kind = "other"
	}
	return category, kind
}

func cleanTitle(value string) string {
	value = strings.ReplaceAll(value, ".", " ")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "(", " ")
	value = strings.ReplaceAll(value, ")", " ")
	value = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(value), " ")
	return value
}

