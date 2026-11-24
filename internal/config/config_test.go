package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var requiredVars = []string{"DB_HOST", "DB_USER", "DB_NAME", "DB_SSLMODE", "REDIS_ADDR"}

var contract = []struct {
	name string
	set  string
	got  func(*Config) string
}{
	{"APP_DEV", "prod", func(c *Config) string { return c.Env }},
	{"STORAGE_DIR", "/tmp/config-test-storage", func(c *Config) string { return c.LocalStorage }},
	{"PORT", "9999", func(c *Config) string { return c.Port }},
	{"SIZE_LIMIT_MB", "77", func(c *Config) string { return strconv.FormatInt(c.SizeLimitMB, 10) }},
	{"ASYNQ_CONCURRENCY", "5", func(c *Config) string { return strconv.Itoa(c.AsynqConcurrency) }},
	{"DB_HOST", "db.example.test", func(c *Config) string { return c.DBHost }},
	{"DB_PORT", "6543", func(c *Config) string { return c.DBPort }},
	{"DB_USER", "svc-user", func(c *Config) string { return c.DBUser }},
	{"DB_PASSWORD", "p@ss /w?rd#1 %41", func(c *Config) string { return c.DBPassword }},
	{"DB_NAME", "video_db", func(c *Config) string { return c.DBName }},
	{"DB_SSLMODE", "verify-full", func(c *Config) string { return c.DBSSLMode }},
	{"REDIS_ADDR", "redis.example.test:6380", func(c *Config) string { return c.RedisAddr }},
	{"REDIS_PASSWORD", "r3d#is p@ss/w?rd", func(c *Config) string { return c.RedisPassword }},
	{"REDIS_DB", "3", func(c *Config) string { return strconv.Itoa(c.RedisDB) }},
}

var retiredVars = []string{
	"DATABASE_URL", "DB_URL", "REDIS_URL",
	"PGHOST", "PGPORT", "PGUSER", "PGPASSWORD", "PGDATABASE", "PGSSLMODE",
	"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
	"DBHOST", "DBPORT", "DBUSER", "DBPASSWORD", "DBNAME", "DBSSLMODE",
	"DBUser", "DBPassword", "DBName",
}

func setContract(t *testing.T, overrides map[string]string) {
	t.Helper()

	for name := range overrides {
		if !contractHas(name) {
			t.Fatalf("override %q is not a variable in the contract table (typo?)", name)
		}
	}

	for _, c := range contract {
		value := c.set
		if override, ok := overrides[c.name]; ok {
			value = override
		}
		t.Setenv(c.name, value)
	}

	for _, name := range retiredVars {
		t.Setenv(name, "")
	}
}

func contractHas(name string) bool {
	for _, c := range contract {
		if c.name == name {
			return true
		}
	}
	return false
}

func TestLoad_EveryVariableReachesItsField(t *testing.T) {
	setContract(t, nil)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error for a complete environment: %v", err)
	}

	for _, c := range contract {
		if got := c.got(cfg); got != c.set {
			t.Errorf("%s landed wrong: field = %q, want %q", c.name, got, c.set)
		}
	}
}

func TestLoad_MissingRequiredVariableFailsAndNamesIt(t *testing.T) {
	for _, name := range requiredVars {
		t.Run(name, func(t *testing.T) {
			setContract(t, map[string]string{name: ""})

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load() succeeded with %s empty; it must refuse to start", name)
			}
			if cfg != nil {
				t.Errorf("Load() returned a non-nil Config alongside an error")
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error does not name the missing variable %s: %q", name, err)
			}
		})
	}
}

func TestLoad_ReportsAllMissingTogether(t *testing.T) {
	setContract(t, map[string]string{"DB_HOST": "", "DB_SSLMODE": "", "REDIS_ADDR": ""})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded with three required variables empty")
	}

	for _, name := range []string{"DB_HOST", "DB_SSLMODE", "REDIS_ADDR"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error omits %s: %q", name, err)
		}
	}
	for _, name := range []string{"DB_USER", "DB_NAME"} {
		if strings.Contains(err.Error(), name) {
			t.Errorf("error names %s, which was provided: %q", name, err)
		}
	}
}

func TestLoad_EmptyPasswordsAreAllowed(t *testing.T) {
	setContract(t, map[string]string{"DB_PASSWORD": "", "REDIS_PASSWORD": ""})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("empty passwords must be allowed, got: %v", err)
	}
	if cfg.DBPassword != "" || cfg.RedisPassword != "" {
		t.Errorf("empty passwords were not preserved: db=%q redis=%q", cfg.DBPassword, cfg.RedisPassword)
	}
}

func TestLoad_Defaults(t *testing.T) {
	overrides := map[string]string{}
	for _, c := range contract {
		overrides[c.name] = ""
	}
	for _, name := range requiredVars {
		overrides[name] = "set-for-this-test"
	}
	setContract(t, overrides)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want the default %q", cfg.Env, "dev")
	}
	if cfg.Port != "3000" {
		t.Errorf("Port = %q, want the default %q", cfg.Port, "3000")
	}
	if cfg.SizeLimitMB != 512 {
		t.Errorf("SizeLimitMB = %d, want the default 512", cfg.SizeLimitMB)
	}
	if cfg.AsynqConcurrency != 2 {
		t.Errorf("AsynqConcurrency = %d, want the default 2", cfg.AsynqConcurrency)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("DBPort = %q, want the default %q", cfg.DBPort, "5432")
	}
	if cfg.RedisDB != 0 {
		t.Errorf("RedisDB = %d, want the default 0", cfg.RedisDB)
	}
	if cfg.DBSSLMode != "set-for-this-test" {
		t.Errorf("DBSSLMode = %q, want the provided value passed through", cfg.DBSSLMode)
	}
	if !filepath.IsAbs(cfg.LocalStorage) {
		t.Errorf("LocalStorage = %q, want an absolute path", cfg.LocalStorage)
	}
}

func TestLoad_InvalidRedisDBIsFatal(t *testing.T) {
	setContract(t, map[string]string{"REDIS_DB": "not-a-number"})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded with a non-numeric REDIS_DB")
	}
	if !strings.Contains(err.Error(), "REDIS_DB") {
		t.Errorf("error does not name REDIS_DB: %q", err)
	}
}

func TestLoad_NumericFallbacks(t *testing.T) {
	setContract(t, map[string]string{"SIZE_LIMIT_MB": "big", "ASYNQ_CONCURRENCY": "many"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.SizeLimitMB != 512 {
		t.Errorf("SizeLimitMB = %d, want the fallback 512", cfg.SizeLimitMB)
	}
	if cfg.AsynqConcurrency != 2 {
		t.Errorf("AsynqConcurrency = %d, want the fallback 2", cfg.AsynqConcurrency)
	}
}

func TestLoad_NoFallbackToRetiredNames(t *testing.T) {
	for _, c := range contract {
		t.Setenv(c.name, "")
	}
	for i, name := range retiredVars {
		t.Setenv(name, "retired-"+strconv.Itoa(i))
	}

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded from retired variable names alone")
	}
	if cfg != nil {
		t.Errorf("Load() returned a Config alongside an error")
	}
	for _, name := range requiredVars {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error omits the live variable %s: %q", name, err)
		}
	}
}

func TestContractCoversEveryVariableLoadReads(t *testing.T) {
	source, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatalf("cannot read config.go: %v", err)
	}

	reads := map[string]bool{}
	pattern := regexp.MustCompile(`(?:Getenv|getEnv)\("([A-Z0-9_]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		reads[match[1]] = true
	}

	for name := range reads {
		if !contractHas(name) {
			t.Errorf("config.go reads %s but the contract table has no entry for it", name)
		}
	}
	for _, c := range contract {
		if !reads[c.name] {
			t.Errorf("the contract table lists %s but config.go no longer reads it", c.name)
		}
	}
}

func TestEnvExampleDocumentsExactlyTheContract(t *testing.T) {
	documented, err := documentedVars(filepath.Join("..", "..", ".env.example"))
	if err != nil {
		t.Fatalf("cannot read .env.example: %v", err)
	}

	for _, c := range contract {
		if !documented[c.name] {
			t.Errorf("%s is read by Load() but missing from .env.example", c.name)
		}
	}
	for name := range documented {
		if !contractHas(name) {
			t.Errorf(".env.example documents %s, which nothing reads", name)
		}
	}
	for _, name := range requiredVars {
		if !documented[name] {
			t.Errorf("required variable %s is missing from .env.example", name)
		}
	}
}

func documentedVars(path string) (map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	names := map[string]bool{}
	line := regexp.MustCompile(`^([A-Z0-9_]+)=`)
	for _, text := range strings.Split(string(raw), "\n") {
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "#"))
		if match := line.FindStringSubmatch(text); match != nil {
			names[match[1]] = true
		}
	}
	return names, nil
}

func TestComposeAppEnvMatchesTheContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("cannot read docker-compose.yml: %v", err)
	}

	key := regexp.MustCompile(`^  ([A-Z0-9_]+):`)
	inBlock := false
	var keys []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "x-app-env:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		match := key.FindStringSubmatch(line)
		if match == nil {
			break
		}
		keys = append(keys, match[1])
	}

	if len(keys) == 0 {
		t.Fatal("no x-app-env variables found in docker-compose.yml")
	}

	for _, name := range keys {
		if !contractHas(name) {
			t.Errorf("x-app-env injects %s, which Load() does not read", name)
		}
	}

	for _, want := range requiredVars {
		found := false
		for _, name := range keys {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("required variable %s is not injected by x-app-env", want)
		}
	}
}

func TestContractHasNoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	var dupes []string
	for _, c := range contract {
		if seen[c.name] {
			dupes = append(dupes, c.name)
		}
		seen[c.name] = true
	}
	if len(dupes) > 0 {
		sort.Strings(dupes)
		t.Errorf("duplicate contract entries: %s", strings.Join(dupes, ", "))
	}
}
