package reorganize

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MoveResult struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Operation  string `json:"operation"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes"`
	RolledBack bool   `json:"rolled_back"`
}

// ExecutePlan performs one approved local copy or move. It never overwrites a
// target, validates the expected hash before the operation and validates the
// resulting file. A failed move is rolled back; a failed copy only removes the
// incomplete target and always preserves the source.
func ExecutePlan(ctx context.Context, plan Plan, expectedSHA256 string) (MoveResult, error) {
	if err := ctx.Err(); err != nil {
		return MoveResult{}, err
	}
	operation := strings.ToLower(strings.TrimSpace(plan.Operation))
	if operation == "" {
		operation = "move"
	}
	if operation != "copy" && operation != "move" {
		return MoveResult{}, fmt.Errorf("unsupported operation %q", plan.Operation)
	}
	if plan.Action != "planned" {
		return MoveResult{}, fmt.Errorf("plan action %q is not executable", plan.Action)
	}
	if plan.Source == "" || plan.TargetFile == "" {
		return MoveResult{}, fmt.Errorf("source and target file are required")
	}
	info, err := os.Stat(plan.Source)
	if err != nil {
		return MoveResult{}, fmt.Errorf("stat source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return MoveResult{}, fmt.Errorf("source is not a regular file")
	}
	sha, bytes, err := fileSHA256(plan.Source)
	if err != nil {
		return MoveResult{}, fmt.Errorf("hash source: %w", err)
	}
	if expectedSHA256 != "" && !strings.EqualFold(sha, expectedSHA256) {
		return MoveResult{}, fmt.Errorf("source hash mismatch: got %s want %s", sha, expectedSHA256)
	}
	if _, err := os.Stat(plan.TargetFile); err == nil {
		return MoveResult{}, fmt.Errorf("target already exists: %s", plan.TargetFile)
	} else if !os.IsNotExist(err) {
		return MoveResult{}, fmt.Errorf("stat target: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return MoveResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(plan.TargetFile), 0750); err != nil {
		return MoveResult{}, fmt.Errorf("create target directory: %w", err)
	}
	if operation == "copy" {
		err = copyFile(ctx, plan.Source, plan.TargetFile, info.Mode())
	} else {
		err = os.Rename(plan.Source, plan.TargetFile)
	}
	if err != nil {
		if operation == "copy" {
			_ = os.Remove(plan.TargetFile)
		}
		return MoveResult{}, fmt.Errorf("%s file: %w", operation, err)
	}
	actual, targetBytes, err := fileSHA256(plan.TargetFile)
	if err == nil && (actual != sha || targetBytes != bytes) {
		err = fmt.Errorf("post-%s validation mismatch: got sha=%s bytes=%d", operation, actual, targetBytes)
	}
	if err != nil {
		rolledBack := false
		if operation == "copy" {
			if rollbackErr := os.Remove(plan.TargetFile); rollbackErr == nil || os.IsNotExist(rollbackErr) {
				rolledBack = true
			}
		} else if rollbackErr := os.Rename(plan.TargetFile, plan.Source); rollbackErr == nil {
			rolledBack = true
		}
		return MoveResult{Source: plan.Source, Target: plan.TargetFile, Operation: operation, SHA256: actual, Bytes: targetBytes, RolledBack: rolledBack}, fmt.Errorf("validate %sed file: %w", operation, err)
	}
	return MoveResult{Source: plan.Source, Target: plan.TargetFile, Operation: operation, SHA256: actual, Bytes: targetBytes}, nil
}

func copyFile(ctx context.Context, source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return ctx.Err()
}

func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", n, err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), n, nil
}

