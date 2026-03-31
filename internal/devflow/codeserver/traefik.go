package codeserver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// TraefikConfig holds settings for dynamic Traefik route generation.
type TraefikConfig struct {
	ConfigDir string // directory Traefik watches (file provider)
	Domain    string // base domain, e.g. "dev.example.com"
}

// RegisterRoute writes a Traefik dynamic config file that routes
// code-{projectSlug}-{envSlug}.{domain} → localhost:{port}
func (t *TraefikConfig) RegisterRoute(projectSlug, envSlug string, port int) error {
	if t == nil || t.ConfigDir == "" || t.Domain == "" {
		return nil // Traefik not configured — skip silently
	}

	routeName := fmt.Sprintf("code-%s-%s", projectSlug, envSlug)
	host := fmt.Sprintf("%s.%s", routeName, t.Domain)

	config := fmt.Sprintf(`http:
  routers:
    %s:
      rule: "Host(\x60%s\x60)"
      service: %s
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
  services:
    %s:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:%d"
`, routeName, host, routeName, routeName, port)

	filename := filepath.Join(t.ConfigDir, routeName+".yml")
	if err := os.MkdirAll(t.ConfigDir, 0755); err != nil {
		return fmt.Errorf("create traefik config dir: %w", err)
	}
	if err := os.WriteFile(filename, []byte(config), 0644); err != nil {
		return fmt.Errorf("write traefik route: %w", err)
	}

	slog.Info("devflow.traefik.route_registered", "host", host, "port", port, "file", filename)
	return nil
}

// RemoveRoute removes the Traefik dynamic config file for a route.
func (t *TraefikConfig) RemoveRoute(projectSlug, envSlug string) error {
	if t == nil || t.ConfigDir == "" {
		return nil
	}

	routeName := fmt.Sprintf("code-%s-%s", projectSlug, envSlug)
	filename := filepath.Join(t.ConfigDir, routeName+".yml")

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove traefik route: %w", err)
	}

	slog.Info("devflow.traefik.route_removed", "route", routeName)
	return nil
}

// RouteURL returns the public URL for a code-server instance.
func (t *TraefikConfig) RouteURL(projectSlug, envSlug string) string {
	if t == nil || t.Domain == "" {
		return ""
	}
	return fmt.Sprintf("https://code-%s-%s.%s", projectSlug, envSlug, t.Domain)
}
