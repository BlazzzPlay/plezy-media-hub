package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/jackc/pgx/v5"
)

var databaseName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func main() {
	adminURL := os.Getenv("MEDIA_OPS_ADMIN_DATABASE_URL")
	name := os.Getenv("MEDIA_OPS_DATABASE_NAME")
	if name == "" {
		name = "plezy_media_hub"
	}
	if adminURL == "" {
		panic("MEDIA_OPS_ADMIN_DATABASE_URL is required")
	}
	if !databaseName.MatchString(name) {
		panic("MEDIA_OPS_DATABASE_NAME must contain only lowercase letters, numbers and underscores")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists); err != nil {
		panic(err)
	}
	if exists {
		fmt.Printf("database %s already exists\n", name)
		return
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		panic(err)
	}
	fmt.Printf("database %s created\n", name)
}
