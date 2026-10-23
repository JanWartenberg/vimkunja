package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setConfigHome(t *testing.T, path string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", path)
	t.Setenv("AppData", path)
}

func TestLoadUserConfigUsesDefaultsWhenMissing(t *testing.T) {
	setConfigHome(t, t.TempDir())

	config, warnings, err := LoadUserConfig(ConfigOverrides{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
	if config.LeaderTimeout != DefaultLeaderTimeout || config.CtrlWTimeout != DefaultCtrlWTimeout {
		t.Fatalf("config = %#v, want built-in defaults", config)
	}
}

func TestLoadUserConfigAppliesSupportedValuesAndWarnsForUnknowns(t *testing.T) {
	configHome := t.TempDir()
	setConfigHome(t, configHome)
	path := filepath.Join(configHome, "vimkunja", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("root = true\n[tui]\nleader_timeout = \"1s\"\nctrl_w_timeout = \"bad\"\nunknown = true\n[workspace]\nproject = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	config, warnings, err := LoadUserConfig(ConfigOverrides{})
	if err != nil {
		t.Fatal(err)
	}
	if config.LeaderTimeout != time.Second || config.CtrlWTimeout != DefaultCtrlWTimeout {
		t.Fatalf("config = %#v, want valid value plus default fallback", config)
	}
	if len(warnings) != 4 || !strings.Contains(strings.Join(warnings, "\n"), "unknown section") || !strings.Contains(strings.Join(warnings, "\n"), "unknown setting root") {
		t.Fatalf("warnings = %#v, want three detailed warnings", warnings)
	}
}

func TestLoadUserConfigCLIOverridesFile(t *testing.T) {
	configHome := t.TempDir()
	setConfigHome(t, configHome)
	path := filepath.Join(configHome, "vimkunja", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[tui]\nleader_timeout = \"1s\"\nctrl_w_timeout = \"2s\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	config, _, err := LoadUserConfig(ConfigOverrides{LeaderTimeout: "3s", CtrlWTimeout: "750ms"})
	if err != nil {
		t.Fatal(err)
	}
	if config.LeaderTimeout != 3*time.Second || config.CtrlWTimeout != 750*time.Millisecond {
		t.Fatalf("config = %#v, want CLI overrides", config)
	}
}

func TestLoadUserConfigRejectsInvalidCLIValue(t *testing.T) {
	_, _, err := LoadUserConfig(ConfigOverrides{LeaderTimeout: "0"})
	if err == nil || !strings.Contains(err.Error(), "--leader-timeout") {
		t.Fatalf("error = %v, want invalid leader timeout error", err)
	}
}

func TestLoadUserConfigRejectsExplicitEmptyCLIValue(t *testing.T) {
	_, _, err := LoadUserConfig(ConfigOverrides{LeaderTimeoutSet: true})
	if err == nil || !strings.Contains(err.Error(), "--leader-timeout") {
		t.Fatalf("error = %v, want invalid empty leader timeout error", err)
	}
}

func TestLoadCodeSettingsAndIndependentFallbacks(t *testing.T) {
	dir := t.TempDir()
	setConfigHome(t, dir)
	path := filepath.Join(dir, "vimkunja", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		text        string
		tab, indent int
		alias       string
		warnings    int
	}{
		{"[tui]\ntabstop=8\nindent_step=2\nliteral_tab_alias='ctrl+b'\n", 8, 2, "ctrl+b", 0},
		{"[tui]\ntabstop=0\nindent_step=3\nliteral_tab_alias='ctrl+w'\n", 4, 4, "ctrl+q", 3},
		{"[tui]\ntabstop=3\nindent_step=4\nliteral_tab_alias=''\n", 3, 4, "", 0},
	} {
		if err := os.WriteFile(path, []byte(tc.text), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, warnings, err := LoadUserConfig(ConfigOverrides{LeaderTimeout: "1s"})
		if err != nil || cfg.Tabstop != tc.tab || cfg.IndentStep != tc.indent || cfg.LiteralTabAlias != tc.alias || len(warnings) != tc.warnings {
			t.Fatalf("%+v %v %v", cfg, warnings, err)
		}
	}
}
