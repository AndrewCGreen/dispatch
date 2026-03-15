// Package config handles loading and validating Dispatch configuration files.
// The main config file (dispatch.yaml) is loaded at startup, and site configs
// are discovered by scanning the sites/ directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level dispatch configuration.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Queue      QueueConfig      `yaml:"queue"`
	Compliance ComplianceConfig `yaml:"compliance"`
	Logging    LoggingConfig    `yaml:"logging"`

	// Loaded at runtime
	Sites map[string]*SiteConfig `yaml:"-"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type AuthConfig struct {
	MasterKey string `yaml:"master_key"`
	SecretKey string `yaml:"secret_key"` // used for signing tokens; falls back to master_key
}


type QueueConfig struct {
	Workers      int    `yaml:"workers"`
	RetryMax     int    `yaml:"retry_max"`
	RetryBackoff string `yaml:"retry_backoff"`
}

type ComplianceConfig struct {
	GDPREnabled       bool `yaml:"gdpr_enabled"`
	ConsentLogging    bool `yaml:"consent_logging"`
	SuppressionGlobal bool `yaml:"suppression_global"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// SiteConfig represents a single site's configuration.
type SiteConfig struct {
	Slug        string            `yaml:"slug"`
	Name        string            `yaml:"name"`
	From        string            `yaml:"from"`
	FromName    string            `yaml:"from_name"`
	ReplyTo     string            `yaml:"reply_to"`
	Backend     string            `yaml:"backend"`
	BackendCfg  map[string]string `yaml:"backend_config"`
	TemplateDir string            `yaml:"templates_dir"`
	DoubleOptin bool              `yaml:"double_optin"`
	OptinTpl    string            `yaml:"optin_template"`
	WelcomeTpl  string            `yaml:"welcome_template"`
	APIKey      string            `yaml:"api_key"`
	Tags        []string          `yaml:"tags"`
	Lists       []ListConfig      `yaml:"lists"`

	// Resolved at load time
	BasePath string `yaml:"-"`
}

type ListConfig struct {
	Slug           string `yaml:"slug"`
	Name           string `yaml:"name"`
	DoubleOptin    bool   `yaml:"double_optin"`
	Unsubscribable *bool  `yaml:"unsubscribable"` // nil = true (default)
}

// Load reads the main config file and discovers all site configs.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "./data/dispatch.db"
	}
	if cfg.Queue.Workers == 0 {
		cfg.Queue.Workers = 4
	}
	if cfg.Queue.RetryMax == 0 {
		cfg.Queue.RetryMax = 3
	}
	if cfg.Queue.RetryBackoff == "" {
		cfg.Queue.RetryBackoff = "5m,30m,2h"
	}

	// Load site configs
	cfg.Sites = make(map[string]*SiteConfig)
	sitesDir := "sites"
	entries, err := os.ReadDir(sitesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading sites directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		siteFile := filepath.Join(sitesDir, entry.Name(), "site.yaml")
		site, err := loadSite(siteFile)
		if err != nil {
			return nil, fmt.Errorf("loading site %s: %w", entry.Name(), err)
		}
		site.BasePath = filepath.Join(sitesDir, entry.Name())
		cfg.Sites[site.Slug] = site
	}

	return &cfg, nil
}

func loadSite(path string) (*SiteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading site config: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var site SiteConfig
	if err := yaml.Unmarshal([]byte(expanded), &site); err != nil {
		return nil, fmt.Errorf("parsing site config: %w", err)
	}

	if site.Slug == "" {
		return nil, fmt.Errorf("site config missing slug")
	}

	// Expand ${VAR} patterns in backend config values
	for k, v := range site.BackendCfg {
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			site.BackendCfg[k] = os.ExpandEnv(v)
		}
	}

	return &site, nil
}

// TokenSecret returns the secret used for signing tokens.
// Falls back to the master key if no dedicated secret_key is configured.
func (c *Config) TokenSecret() string {
	if c.Auth.SecretKey != "" {
		return c.Auth.SecretKey
	}
	return c.Auth.MasterKey
}

// GetSite returns the site config for a given slug, or an error if not found.
func (c *Config) GetSite(slug string) (*SiteConfig, error) {
	site, ok := c.Sites[slug]
	if !ok {
		return nil, fmt.Errorf("site %q not found", slug)
	}
	return site, nil
}

// TemplatesPath returns the resolved templates directory for a site.
func (s *SiteConfig) TemplatesPath() string {
	if s.TemplateDir == "" {
		return filepath.Join(s.BasePath, "templates")
	}
	if filepath.IsAbs(s.TemplateDir) {
		return s.TemplateDir
	}
	return filepath.Join(s.BasePath, s.TemplateDir)
}
