package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/naming"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct{ db *sql.DB }

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	s := &Store{db: db}
	if err := s.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	for version := 1; version <= 10; version++ {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sqlBytes, err := migrations.ReadFile(fmt.Sprintf("migrations/%03d_%s.sql", version, map[int]string{1: "catalog", 2: "inventory", 3: "parsed_inventory", 4: "identity_versions", 5: "technical_preference", 6: "workflow", 7: "candidate_details", 8: "candidate_approval", 9: "reorganization_integrity", 10: "copy_operation"}[version]))
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

type Content struct {
	ID         int64  `json:"id"`
	Category   string `json:"category"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Year       *int   `json:"year,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
}

type InventoryFile struct {
	ID           int64             `json:"id,omitempty"`
	Path         string            `json:"path"`
	RelativePath string            `json:"relative_path"`
	Size         int64             `json:"size"`
	ModifiedAt   time.Time         `json:"modified_at"`
	SHA256       string            `json:"sha256"`
	Container    string            `json:"container,omitempty"`
	VideoCodec   string            `json:"video_codec,omitempty"`
	Width        int               `json:"width,omitempty"`
	Height       int               `json:"height,omitempty"`
	Duration     float64           `json:"duration,omitempty"`
	HDR          string            `json:"hdr,omitempty"`
	Bitrate      int64             `json:"bitrate,omitempty"`
	Parsed       naming.ParsedName `json:"parsed"`
}

func (s *Store) ListContents(ctx context.Context, category string, limit int) ([]Content, error) {
	query := `SELECT id, category, kind, title, year, COALESCE(external_id,'') FROM contents`
	args := []any{}
	if category != "" {
		query += ` WHERE category=$1`
		args = append(args, category)
	}
	query += fmt.Sprintf(` ORDER BY title LIMIT %d`, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Content, 0)
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.ID, &c.Category, &c.Kind, &c.Title, &c.Year, &c.ExternalID); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *Store) CreateContent(ctx context.Context, c Content) (Content, error) {
	err := s.db.QueryRowContext(ctx, `INSERT INTO contents(category,kind,title,year,external_id) VALUES($1,$2,$3,$4,NULLIF($5,'')) RETURNING id`, c.Category, c.Kind, c.Title, c.Year, c.ExternalID).Scan(&c.ID)
	return c, err
}

func (s *Store) SaveInventory(ctx context.Context, f InventoryFile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inventory_files(path, relative_path, size_bytes, modified_at, sha256, container, video_codec, width, height, duration_seconds, category, kind, parsed_title, parsed_year, season, episode, edition, source, audio, hdr, bitrate)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,0),NULLIF($9,0),NULLIF($10,0),$11,$12,NULLIF($13,''),$14,$15,$16,NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),NULLIF($20,''),NULLIF($21,0))
		ON CONFLICT (path) DO UPDATE SET relative_path=EXCLUDED.relative_path, size_bytes=EXCLUDED.size_bytes,
		modified_at=EXCLUDED.modified_at, sha256=EXCLUDED.sha256, container=EXCLUDED.container,
		video_codec=EXCLUDED.video_codec, width=EXCLUDED.width, height=EXCLUDED.height,
		duration_seconds=EXCLUDED.duration_seconds, category=EXCLUDED.category, kind=EXCLUDED.kind,
		parsed_title=EXCLUDED.parsed_title, parsed_year=EXCLUDED.parsed_year, season=EXCLUDED.season,
		episode=EXCLUDED.episode, edition=EXCLUDED.edition, source=EXCLUDED.source, audio=EXCLUDED.audio, hdr=EXCLUDED.hdr, bitrate=EXCLUDED.bitrate, updated_at=now()`,
		f.Path, f.RelativePath, f.Size, f.ModifiedAt, f.SHA256, f.Container, f.VideoCodec, f.Width, f.Height, f.Duration,
		f.Parsed.Category, f.Parsed.Kind, f.Parsed.Title, f.Parsed.Year, f.Parsed.Season, f.Parsed.Episode,
		f.Parsed.Edition, f.Parsed.Source, f.Parsed.Audio, f.HDR, f.Bitrate)
	return err
}

// SaveParsedCandidate records a local filename interpretation without pretending it is a provider match.
func (s *Store) SaveParsedCandidate(ctx context.Context, path, title string, year *int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO identity_candidates(inventory_file_id, provider, title, year)
		SELECT id, 'manual', $2, $3 FROM inventory_files WHERE path=$1
		ON CONFLICT DO NOTHING`, path, title, year)
	return err
}

func (s *Store) ApprovedExternalID(ctx context.Context, path string) (string, error) {
	var externalID string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(external_id,'') FROM inventory_files WHERE path=$1 AND identity_status='identified'`, path).Scan(&externalID)
	return externalID, err
}

type PlanInput struct {
	Path       string
	Category   string
	Kind       string
	Title      string
	Year       *int
	ExternalID string
}

func (s *Store) LoadPlanInput(ctx context.Context, path string) (PlanInput, error) {
	var p PlanInput
	err := s.db.QueryRowContext(ctx, `SELECT path,category,kind,parsed_title,parsed_year,COALESCE(external_id,'') FROM inventory_files WHERE path=$1`, path).Scan(&p.Path, &p.Category, &p.Kind, &p.Title, &p.Year, &p.ExternalID)
	return p, err
}

type DuplicateGroup struct {
	SHA256 string   `json:"sha256"`
	Count  int      `json:"count"`
	Paths  []string `json:"paths"`
}

type ReorganizationPlan struct {
	ID         int64  `json:"id"`
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Operation  string `json:"operation"`
	Action     string `json:"action"`
	Reason     string `json:"reason"`
	SHA256     string `json:"sha256,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
}

func (s *Store) ListReorganizationPlans(ctx context.Context, action string, limit int) ([]ReorganizationPlan, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,source_path,COALESCE(target_path,''),COALESCE(operation,'move'),action,reason,COALESCE(source_sha256,''),COALESCE(bytes,0) FROM reorganization_plans WHERE ($1='' OR action=$1) ORDER BY id LIMIT $2`, action, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := make([]ReorganizationPlan, 0)
	for rows.Next() {
		var p ReorganizationPlan
		if err := rows.Scan(&p.ID, &p.SourcePath, &p.TargetPath, &p.Operation, &p.Action, &p.Reason, &p.SHA256, &p.Bytes); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func (s *Store) LoadReorganizationPlanByID(ctx context.Context, id int64) (ReorganizationPlan, error) {
	var p ReorganizationPlan
	err := s.db.QueryRowContext(ctx, `SELECT id,source_path,COALESCE(target_path,''),COALESCE(operation,'move'),action,reason,COALESCE(source_sha256,''),COALESCE(bytes,0) FROM reorganization_plans WHERE id=$1`, id).Scan(&p.ID, &p.SourcePath, &p.TargetPath, &p.Operation, &p.Action, &p.Reason, &p.SHA256, &p.Bytes)
	return p, err
}

type EnrichedCandidate struct {
	Provider       string
	ExternalID     string
	Title          string
	Year           *int
	Genres         []string
	Director       string
	Actors         []string
	Rating         float64
	RuntimeMinutes int
	Country        string
	Language       string
	Certification  string
	Tags           []string
	Edition        string
}

type Candidate struct {
	ID              int64    `json:"id"`
	InventoryFileID int64    `json:"inventory_file_id"`
	Path            string   `json:"path"`
	Provider        string   `json:"provider"`
	ExternalID      string   `json:"external_id,omitempty"`
	Title           string   `json:"title"`
	Year            *int     `json:"year,omitempty"`
	Genres          []string `json:"genres,omitempty"`
	Director        string   `json:"director,omitempty"`
	Actors          []string `json:"actors,omitempty"`
	Rating          float64  `json:"rating,omitempty"`
	RuntimeMinutes  int      `json:"runtime_minutes,omitempty"`
	Country         string   `json:"country,omitempty"`
	Language        string   `json:"language,omitempty"`
	Certification   string   `json:"certification,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Edition         string   `json:"edition,omitempty"`
	Status          string   `json:"status"`
}

func (s *Store) ListCandidates(ctx context.Context, status string, limit int) ([]Candidate, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.inventory_file_id,i.path,c.provider,COALESCE(c.external_id,''),c.title,c.year,c.genres,COALESCE(c.director,''),c.actors,COALESCE(c.rating,0)::double precision,COALESCE(c.runtime_minutes,0),COALESCE(c.country,''),COALESCE(c.language,''),COALESCE(c.certification,''),c.tags,COALESCE(c.edition,''),c.status FROM identity_candidates c JOIN inventory_files i ON i.id=c.inventory_file_id WHERE ($1='' OR c.status=$1) ORDER BY c.id LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Candidate, 0)
	for rows.Next() {
		var c Candidate
		var genres, actors, tags []byte
		if err := rows.Scan(&c.ID, &c.InventoryFileID, &c.Path, &c.Provider, &c.ExternalID, &c.Title, &c.Year, &genres, &c.Director, &actors, &c.Rating, &c.RuntimeMinutes, &c.Country, &c.Language, &c.Certification, &tags, &c.Edition, &c.Status); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(genres, &c.Genres)
		_ = json.Unmarshal(actors, &c.Actors)
		_ = json.Unmarshal(tags, &c.Tags)
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *Store) ApproveCandidate(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var fileID int64
	var category, kind, title, provider, externalID string
	var year *int
	if err = tx.QueryRowContext(ctx, `SELECT c.inventory_file_id,i.category,i.kind,c.title,c.year,c.provider,COALESCE(c.external_id,'') FROM identity_candidates c JOIN inventory_files i ON i.id=c.inventory_file_id WHERE c.id=$1`, id).Scan(&fileID, &category, &kind, &title, &year, &provider, &externalID); err != nil {
		return err
	}
	contentExternal := provider + ":" + externalID
	var contentID int64
	if _, err = tx.ExecContext(ctx, `INSERT INTO contents(category,kind,title,year,external_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, category, kind, title, year, contentExternal); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT id FROM contents WHERE category=$1 AND kind=$2 AND title=$3 AND year IS NOT DISTINCT FROM $4 AND external_id=$5`, category, kind, title, year, contentExternal).Scan(&contentID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE identity_candidates SET status='rejected' WHERE inventory_file_id=$1 AND id<>$2 AND status='pending'`, fileID, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE identity_candidates SET status='accepted' WHERE id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE inventory_files SET identity_status='identified',external_id=$2,content_id=$3 WHERE id=$1`, fileID, contentExternal, contentID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) RejectCandidate(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE identity_candidates SET status='rejected' WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SaveEnrichedCandidate(ctx context.Context, path string, c EnrichedCandidate) error {
	provider := normalizeProvider(c.Provider)
	genres, err := json.Marshal(c.Genres)
	if err != nil {
		return err
	}
	actors, err := json.Marshal(c.Actors)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(c.Tags)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO identity_candidates(inventory_file_id,provider,external_id,title,year,genres,director,actors,rating,runtime_minutes,country,language,certification,tags,edition)
		SELECT id,$2,NULLIF($3,''),$4,$5,$6::jsonb,NULLIF($7,''),$8::jsonb,NULLIF($9::numeric,0),NULLIF($10::integer,0),NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),$14::jsonb,NULLIF($15,'')
		FROM inventory_files WHERE path=$1
		ON CONFLICT(inventory_file_id,provider,external_id) DO UPDATE SET title=EXCLUDED.title,year=EXCLUDED.year,genres=EXCLUDED.genres,director=EXCLUDED.director,actors=EXCLUDED.actors,rating=EXCLUDED.rating,runtime_minutes=EXCLUDED.runtime_minutes,country=EXCLUDED.country,language=EXCLUDED.language,certification=EXCLUDED.certification,tags=EXCLUDED.tags,edition=EXCLUDED.edition`,
		path, provider, c.ExternalID, c.Title, c.Year, genres, c.Director, actors, c.Rating, c.RuntimeMinutes, c.Country, c.Language, c.Certification, tags, c.Edition)
	return err
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "themoviedb", "tmdb":
		return "tmdb"
	case "thetvdb", "tvdb":
		return "tvdb"
	case "imdb":
		return "imdb"
	case "anidb":
		return "filebot"
	case "filebot":
		return "filebot"
	default:
		return "manual"
	}
}

func (s *Store) SaveReorganizationPlan(ctx context.Context, p ReorganizationPlan) error {
	operation := p.Operation
	if operation == "" {
		operation = "move"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO reorganization_plans(source_path,target_path,operation,action,reason,source_sha256,bytes) VALUES($1,NULLIF($2,''),$3,$4,$5,NULLIF($6,''),NULLIF($7,0)) ON CONFLICT(source_path) DO UPDATE SET target_path=EXCLUDED.target_path,operation=EXCLUDED.operation,action=EXCLUDED.action,reason=EXCLUDED.reason,source_sha256=EXCLUDED.source_sha256,bytes=EXCLUDED.bytes,updated_at=now()`, p.SourcePath, p.TargetPath, operation, p.Action, p.Reason, p.SHA256, p.Bytes)
	return err
}

func (s *Store) LoadReorganizationPlan(ctx context.Context, source string) (ReorganizationPlan, error) {
	var p ReorganizationPlan
	err := s.db.QueryRowContext(ctx, `SELECT source_path,COALESCE(target_path,''),COALESCE(operation,'move'),action,reason,COALESCE(source_sha256,''),COALESCE(bytes,0) FROM reorganization_plans WHERE source_path=$1`, source).Scan(&p.SourcePath, &p.TargetPath, &p.Operation, &p.Action, &p.Reason, &p.SHA256, &p.Bytes)
	return p, err
}

func (s *Store) CompleteReorganizationPlan(ctx context.Context, source, target, sha256 string, bytes int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE reorganization_plans SET target_path=$2,action='completed',reason='file operation and SHA-256 verification completed',source_sha256=$3,bytes=$4,updated_at=now() WHERE source_path=$1 AND action='planned'`, source, target, sha256, bytes)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) FailReorganizationPlan(ctx context.Context, source, reason string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE reorganization_plans SET action='failed',reason=$2,updated_at=now() WHERE source_path=$1 AND action='planned'`, source, reason)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type Job struct {
	ID          int64           `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	Checkpoint  json.RawMessage `json:"checkpoint"`
	LastError   string          `json:"last_error,omitempty"`
}

func (s *Store) ListJobs(ctx context.Context, status string, limit int) ([]Job, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,job_type,payload,status,attempts,max_attempts,checkpoint,COALESCE(last_error,'') FROM workflow_jobs WHERE ($1='' OR status=$1) ORDER BY id DESC LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]Job, 0)
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Payload, &j.Status, &j.Attempts, &j.MaxAttempts, &j.Checkpoint, &j.LastError); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Store) EnqueueJob(ctx context.Context, jobType string, payload json.RawMessage, maxAttempts int) (int64, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO workflow_jobs(job_type,payload,max_attempts) VALUES($1,$2,$3) RETURNING id`, jobType, payload, maxAttempts).Scan(&id)
	return id, err
}
func (s *Store) ClaimJob(ctx context.Context, worker, jobType string) (*Job, error) {
	var j Job
	err := s.db.QueryRowContext(ctx, `WITH candidate AS (SELECT id FROM workflow_jobs WHERE status='pending' AND next_run_at<=now() AND ($2='' OR job_type=$2) ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE workflow_jobs w SET status='running',locked_by=$1,heartbeat_at=now(),attempts=attempts+1,updated_at=now() FROM candidate WHERE w.id=candidate.id RETURNING w.id,w.job_type,w.payload,w.status,w.attempts,w.max_attempts,w.checkpoint,COALESCE(w.last_error,'')`, worker, jobType).Scan(&j.ID, &j.Type, &j.Payload, &j.Status, &j.Attempts, &j.MaxAttempts, &j.Checkpoint, &j.LastError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &j, err
}
func (s *Store) HeartbeatJob(ctx context.Context, id int64, worker string, checkpoint json.RawMessage) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workflow_jobs SET heartbeat_at=now(),checkpoint=$3,updated_at=now() WHERE id=$1 AND locked_by=$2 AND status='running'`, id, worker, checkpoint)
	return err
}
func (s *Store) CompleteJob(ctx context.Context, id int64, worker string, checkpoint json.RawMessage) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workflow_jobs SET status='completed',checkpoint=$3,locked_by=NULL,heartbeat_at=NULL,updated_at=now() WHERE id=$1 AND locked_by=$2 AND status='running'`, id, worker, checkpoint)
	return err
}
func (s *Store) FailJob(ctx context.Context, id int64, worker string, jobErr error) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workflow_jobs SET status=CASE WHEN attempts<max_attempts THEN 'pending' ELSE 'failed' END,last_error=$3,locked_by=NULL,heartbeat_at=NULL,next_run_at=CASE WHEN attempts<max_attempts THEN now()+interval '30 seconds' ELSE next_run_at END,updated_at=now() WHERE id=$1 AND locked_by=$2 AND status='running'`, id, worker, jobErr.Error())
	return err
}
func (s *Store) RecoverStaleJobs(ctx context.Context, stale time.Duration) error {
	_, err := s.db.ExecContext(ctx, `UPDATE workflow_jobs SET status='pending',locked_by=NULL,heartbeat_at=NULL,last_error=COALESCE(last_error,'')||' [recovered stale job]',updated_at=now() WHERE status='running' AND heartbeat_at < now()-$1::interval`, fmt.Sprintf("%d seconds", int(stale.Seconds())))
	return err
}

func (s *Store) ListExactDuplicates(ctx context.Context) ([]DuplicateGroup, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sha256, count(*), array_agg(path ORDER BY path) FROM inventory_files GROUP BY sha256 HAVING count(*) > 1 ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DuplicateGroup, 0)
	for rows.Next() {
		var group DuplicateGroup
		if err := rows.Scan(&group.SHA256, &group.Count, &group.Paths); err != nil {
			return nil, err
		}
		result = append(result, group)
	}
	return result, rows.Err()
}

