package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/naming"
)

var mediaExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true,
	".ts": true, ".m2ts": true, ".webm": true, ".mp3": true, ".flac": true,
}

type TechnicalMetadata struct {
	Container  string  `json:"container,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
	HDR        string  `json:"hdr,omitempty"`
	Bitrate    int64   `json:"bitrate,omitempty"`
}

type FileRecord struct {
	Path         string
	RelativePath string
	Size         int64
	ModifiedAt   time.Time
	SHA256       string
	TechnicalMetadata
	Parsed naming.ParsedName
}

type Probe func(context.Context, string) (TechnicalMetadata, error)

func Scan(ctx context.Context, root string, probe Probe, maxFiles int) ([]FileRecord, error) {
	if root == "" {
		return nil, fmt.Errorf("scan root is required")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat scan root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root is not a directory")
	}
	if probe == nil {
		probe = FFprobe
	}
	result := make([]FileRecord, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || !mediaExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}
		if maxFiles > 0 && len(result) >= maxFiles {
			return filepath.SkipAll
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		hash, err := SHA256(path)
		if err != nil {
			return err
		}
		technical, err := probe(ctx, path)
		if err != nil {
			return fmt.Errorf("probe %s: %w", path, err)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result = append(result, FileRecord{Path: path, RelativePath: relative, Size: fileInfo.Size(), ModifiedAt: fileInfo.ModTime(), SHA256: hash, TechnicalMetadata: technical, Parsed: naming.Parse(path, root)})
		return nil
	})
	return result, err
}

func SHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func FFprobe(ctx context.Context, path string) (TechnicalMetadata, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "format=format_name,duration,bit_rate:stream=codec_name,width,height,color_transfer", "-of", "json", path)
	output, err := cmd.Output()
	if err != nil {
		return TechnicalMetadata{}, fmt.Errorf("ffprobe: %w", err)
	}
	var raw struct {
		Streams []struct {
			Codec    string `json:"codec_name"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Transfer string `json:"color_transfer"`
		} `json:"streams"`
		Format struct {
			Name     string `json:"format_name"`
			Duration string `json:"duration"`
			Bitrate  string `json:"bit_rate"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return TechnicalMetadata{}, err
	}
	var duration float64
	if raw.Format.Duration != "" {
		_, _ = fmt.Sscan(raw.Format.Duration, &duration)
	}
	var bitrate int64
	if raw.Format.Bitrate != "" {
		_, _ = fmt.Sscan(raw.Format.Bitrate, &bitrate)
	}
	metadata := TechnicalMetadata{Container: raw.Format.Name, Duration: duration, Bitrate: bitrate}
	if len(raw.Streams) > 0 {
		metadata.VideoCodec = raw.Streams[0].Codec
		metadata.Width = raw.Streams[0].Width
		metadata.Height = raw.Streams[0].Height
		switch raw.Streams[0].Transfer {
		case "smpte2084":
			metadata.HDR = "HDR10"
		case "arib-std-b67":
			metadata.HDR = "HLG"
		}
	}
	return metadata, nil
}

