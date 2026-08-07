package identity

import (
	"regexp"
	"strings"
	"unicode"
)

type Candidate struct {
	Provider string
	ID       string
	Title    string
	Year     *int
	Score    float64
}
type Decision struct {
	Candidate   Candidate
	AutoApprove bool
	Reason      string
}

func Select(title string, year *int, candidates []Candidate) Decision {
	if len(candidates) == 0 {
		return Decision{Reason: "no candidates"}
	}
	best := candidates[0]
	second := -1.0
	bestScore := -1.0
	for _, candidate := range candidates {
		score := candidate.Score
		if normalize(candidate.Title) == normalize(title) {
			score += 70
		}
		if year != nil && candidate.Year != nil && *year == *candidate.Year {
			score += 25
		}
		if strings.EqualFold(candidate.Provider, "tmdb") {
			score += 5
		}
		if score > bestScore {
			second = bestScore
			bestScore = score
			best = candidate
		} else if score > second {
			second = score
		}
	}
	decision := Decision{Candidate: best}
	if normalize(best.Title) == normalize(title) && year != nil && best.Year != nil && *year == *best.Year && bestScore-second >= 15 {
		decision.AutoApprove = true
		decision.Reason = "exact title/year with sufficient margin"
	} else {
		decision.Reason = "ambiguous or incomplete match; manual review required"
	}
	return decision
}
func normalize(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return regexp.MustCompile(`\s+`).ReplaceAllString(b.String(), "")
}

