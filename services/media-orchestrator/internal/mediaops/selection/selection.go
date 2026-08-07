package selection

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type Version struct {
	ID         int64
	Container  string
	Codec      string
	Width      int
	Height     int
	HDR        string
	Bitrate    int64
	SHA256     string
	Path       string
	ModifiedAt time.Time
}

func Better(a, b Version) bool {
	ra, rb := rank(a), rank(b)
	for i := range ra {
		if ra[i] != rb[i] {
			return ra[i] > rb[i]
		}
	}
	if !a.ModifiedAt.Equal(b.ModifiedAt) {
		return a.ModifiedAt.After(b.ModifiedAt)
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.ID < b.ID
}
func Choose(versions []Version) (Version, bool) {
	if len(versions) == 0 {
		return Version{}, false
	}
	best := versions[0]
	for _, candidate := range versions[1:] {
		if Better(candidate, best) {
			best = candidate
		}
	}
	return best, true
}

// Deduplicate removes exact SHA-256 duplicates, keeping the newest file and
// using the path as a deterministic tie-breaker. Different encodes remain in
// the result so Choose can apply the technical preference rules.
func Deduplicate(versions []Version) []Version {
	byHash := make(map[string]Version, len(versions))
	withoutHash := make([]Version, 0)
	for _, version := range versions {
		if strings.TrimSpace(version.SHA256) == "" {
			withoutHash = append(withoutHash, version)
			continue
		}
		current, exists := byHash[version.SHA256]
		if !exists || Better(version, current) {
			byHash[version.SHA256] = version
		}
	}
	result := append(withoutHash, values(byHash)...)
	sort.Slice(result, func(i, j int) bool { return Better(result[i], result[j]) })
	return result
}

func Explain(v Version) string {
	codec := normalizeCodec(v.Codec)
	container := normalizeContainer(v.Container)
	hdr := strings.ToUpper(strings.TrimSpace(v.HDR))
	if hdr == "" {
		hdr = "SDR"
	}
	return "1080p-doel=" + resolutionLabel(v.Height) + ", codec=" + codec + ", container=" + container + ", hdr=" + hdr
}

func values(m map[string]Version) []Version {
	result := make([]Version, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
func rank(v Version) [6]int {
	resolution := 0
	switch {
	case v.Height == 1080:
		resolution = 300
	case v.Height > 0 && v.Height < 1080:
		resolution = 200 + v.Height/10
	case v.Height > 1080:
		resolution = 100 - (v.Height-1080)/10
	}
	codec := 0
	switch normalizeCodec(v.Codec) {
	case "hevc", "h265", "x265":
		codec = 30
	case "av1":
		codec = 25
	case "h264", "x264":
		codec = 20
	}
	container := 0
	if normalizeContainer(v.Container) == "mkv" {
		container = 20
	}
	hdr := 0
	switch strings.ToUpper(strings.TrimSpace(v.HDR)) {
	case "HDR10+":
		hdr = 30
	case "HDR10":
		hdr = 20
	case "HLG":
		hdr = 10
	}
	return [6]int{resolution, codec, container, hdr, int(v.Bitrate / 1000000), 0}
}

func normalizeCodec(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hevc", "h265", "x265":
		return "hevc"
	case "h264", "x264", "avc":
		return "h264"
	case "av1":
		return "av1"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeContainer(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "mkv", "matroska", "matroska,webm":
		return "mkv"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func resolutionLabel(height int) string {
	if height <= 0 {
		return "desconocida"
	}
	return formatHeight(height) + "px"
}

func formatHeight(height int) string {
	if height == 1080 {
		return "1080"
	}
	return strconv.Itoa(height)
}

