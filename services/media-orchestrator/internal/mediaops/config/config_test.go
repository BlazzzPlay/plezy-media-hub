package config

import "testing"

func TestLoadRequiresMVPConfiguration(t *testing.T) {
	t.Setenv("PRA_DATABASE_URL", "postgres://local")
	t.Setenv("PRA_MEDIA_CONFIG_ROOT", `C:\Plex\media\.media-config`)
	t.Setenv("PRA_STAGING_ROOT", `C:\Plex\media`)
	t.Setenv("PRA_PORT", "8094")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != 8094 {
		t.Fatalf("port = %d, want 8094", c.Port)
	}
}

