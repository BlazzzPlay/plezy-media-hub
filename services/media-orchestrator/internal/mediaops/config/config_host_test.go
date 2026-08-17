package config

import "testing"

func TestLoadUsesLoopbackByDefault(t *testing.T) {
  t.Setenv("MEDIA_OPS_DATABASE_URL", "postgres://local")
  t.Setenv("MEDIA_OPS_CONFIG_ROOT", `C:\Plex\media\.media-config`)
  t.Setenv("MEDIA_OPS_STAGING_ROOT", `C:\Plex\media`)
  t.Setenv("MEDIA_OPS_HOST", "")
  c, err := Load()
  if err != nil { t.Fatal(err) }
  if c.Host != "127.0.0.1" { t.Fatalf("host = %q, want loopback", c.Host) }
}

func TestLoadAcceptsExplicitTailscaleHost(t *testing.T) {
  t.Setenv("MEDIA_OPS_DATABASE_URL", "postgres://local")
  t.Setenv("MEDIA_OPS_CONFIG_ROOT", `C:\Plex\media\.media-config`)
  t.Setenv("MEDIA_OPS_STAGING_ROOT", `C:\Plex\media`)
  t.Setenv("MEDIA_OPS_HOST", "100.87.101.20")
  c, err := Load()
  if err != nil { t.Fatal(err) }
  if c.Host != "100.87.101.20" { t.Fatalf("host = %q, want Tailscale address", c.Host) }
}
