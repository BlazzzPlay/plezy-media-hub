package config

import "testing"

func TestLoadRequiresMVPConfiguration(t *testing.T) {
	t.Setenv("MEDIA_OPS_DATABASE_URL", "postgres://local")
	t.Setenv("MEDIA_OPS_CONFIG_ROOT", `C:\Plex\media\.media-config`)
	t.Setenv("MEDIA_OPS_STAGING_ROOT", `C:\Plex\media`)
	t.Setenv("MEDIA_OPS_PORT", "8101")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != 8101 {
		t.Fatalf("port = %d, want 8101", c.Port)
	}
}
