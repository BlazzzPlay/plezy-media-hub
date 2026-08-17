package identity

import "testing"

func TestSelectExactCandidate(t *testing.T) {
	year := 2026
	other := 1995
	d := Select("Mortal Kombat II", &year, []Candidate{{Provider: "tmdb", ID: "931285", Title: "Mortal Kombat II", Year: &year}, {Provider: "tmdb", ID: "9312", Title: "Mortal Kombat", Year: &other}})
	if !d.AutoApprove || d.Candidate.ID != "931285" {
		t.Fatalf("decision=%#v", d)
	}
}
func TestSelectAmbiguousRequiresReview(t *testing.T) {
	year := 2026
	d := Select("Unknown", &year, []Candidate{{Provider: "tmdb", ID: "1", Title: "Unknown", Year: &year, Score: 80}, {Provider: "tmdb", ID: "2", Title: "Unknown", Year: &year, Score: 75}})
	if d.AutoApprove {
		t.Fatal("ambiguous candidate auto-approved")
	}
}

