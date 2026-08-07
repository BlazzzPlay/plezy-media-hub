package selection

import (
	"testing"
	"time"
)

func TestChoosePrefers1080HEVCMKV(t *testing.T) {
	got, ok := Choose([]Version{{ID: 1, Container: "mp4", Codec: "h264", Height: 2160, Bitrate: 20000000}, {ID: 2, Container: "matroska", Codec: "hevc", Height: 1080, HDR: "HDR10", Bitrate: 9000000}, {ID: 3, Container: "matroska", Codec: "hevc", Height: 720, Bitrate: 12000000}})
	if !ok || got.ID != 2 {
		t.Fatalf("selected %#v", got)
	}
}

func TestDeduplicateKeepsNewestExactCopy(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()
	got := Deduplicate([]Version{{ID: 1, SHA256: "same", Path: "old.mkv", ModifiedAt: old}, {ID: 2, SHA256: "same", Path: "new.mkv", ModifiedAt: newer}, {ID: 3, SHA256: "other", Path: "other.mkv"}})
	if len(got) != 2 {
		t.Fatalf("got %d versions: %#v", len(got), got)
	}
	for _, v := range got {
		if v.SHA256 == "same" && v.ID != 2 {
			t.Fatalf("kept wrong duplicate: %#v", v)
		}
	}
}

