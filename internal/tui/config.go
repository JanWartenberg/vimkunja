package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultLeaderTimeout and DefaultCtrlWTimeout are the built-in interaction
// timeouts used when no user setting is available.
const (
	DefaultLeaderTimeout = 500 * time.Millisecond
	DefaultCtrlWTimeout  = 2 * time.Second
)

// ConfigOverrides contains values supplied explicitly on the command line.
// Empty values mean that the corresponding flag was not supplied.
type ConfigOverrides struct {
	LeaderTimeout    string
	CtrlWTimeout     string
	LeaderTimeoutSet bool
	CtrlWTimeoutSet  bool
}

// LoadUserConfig loads local TUI settings and applies command-line overrides.
// Recoverable file problems are returned as warnings. Invalid overrides are
// returned as errors because they are explicit user input.
func LoadUserConfig(overrides ConfigOverrides) (Config, []string, error) {
	leader, ctrlW := DefaultLeaderTimeout, DefaultCtrlWTimeout
	warnings := []string{}
	code := Config{Tabstop: 4, IndentStep: 4, LiteralTabAlias: "ctrl+q"}

	configDir, err := os.UserConfigDir()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cannot determine user config directory: %v", err))
	} else {
		path := filepath.Join(configDir, "vimkunja", "config.toml")
		fileLeader, fileCtrlW, fileCode, fileWarnings := loadTOMLConfig(path)
		code = fileCode
		warnings = append(warnings, fileWarnings...)
		if fileLeader != nil {
			leader = *fileLeader
		}
		if fileCtrlW != nil {
			ctrlW = *fileCtrlW
		}
	}

	if overrides.LeaderTimeoutSet || overrides.LeaderTimeout != "" {
		value, err := parsePositiveDuration("--leader-timeout", overrides.LeaderTimeout)
		if err != nil {
			return Config{}, warnings, err
		}
		leader = value
	}
	if overrides.CtrlWTimeoutSet || overrides.CtrlWTimeout != "" {
		value, err := parsePositiveDuration("--ctrl-w-timeout", overrides.CtrlWTimeout)
		if err != nil {
			return Config{}, warnings, err
		}
		ctrlW = value
	}

	code.LeaderTimeout, code.CtrlWTimeout, code.StartupWarnings = leader, ctrlW, warnings
	return code, warnings, nil
}

func loadTOMLConfig(path string) (*time.Duration, *time.Duration, Config, []string) {
	code := Config{Tabstop: 4, IndentStep: 4, LiteralTabAlias: "ctrl+q"}
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, code, nil
	}
	if err != nil {
		return nil, nil, code, []string{fmt.Sprintf("cannot read config file %s: %v", path, err)}
	}

	var document map[string]interface{}
	if _, err := toml.Decode(string(contents), &document); err != nil {
		return nil, nil, code, []string{fmt.Sprintf("cannot parse config file %s: %v", path, err)}
	}

	warnings := []string{}
	leader, ctrlW := (*time.Duration)(nil), (*time.Duration)(nil)
	topLevelKeys := make([]string, 0, len(document))
	for name := range document {
		topLevelKeys = append(topLevelKeys, name)
	}
	sort.Strings(topLevelKeys)
	for _, name := range topLevelKeys {
		raw := document[name]
		if name != "tui" {
			if _, isSection := raw.(map[string]interface{}); isSection {
				warnings = append(warnings, fmt.Sprintf("config file %s: unknown section [%s] ignored", path, name))
			} else {
				warnings = append(warnings, fmt.Sprintf("config file %s: unknown setting %s ignored", path, name))
			}
			continue
		}
		section, ok := raw.(map[string]interface{})
		if !ok {
			warnings = append(warnings, fmt.Sprintf("config file %s: [tui] must be a table", path))
			continue
		}
		sectionKeys := make([]string, 0, len(section))
		for key := range section {
			sectionKeys = append(sectionKeys, key)
		}
		sort.Strings(sectionKeys)
		for _, key := range sectionKeys {
			value := section[key]
			switch key {
			case "leader_timeout":
				duration, warning := configDuration(path, "tui.leader_timeout", value)
				warnings = append(warnings, warning...)
				if duration != nil {
					leader = duration
				}
			case "ctrl_w_timeout":
				duration, warning := configDuration(path, "tui.ctrl_w_timeout", value)
				warnings = append(warnings, warning...)
				if duration != nil {
					ctrlW = duration
				}
			case "tabstop", "indent_step":
				number, ok := value.(int64)
				if !ok || number <= 0 || key == "indent_step" && number != 2 && number != 4 || int64(int(number)) != number {
					warnings = append(warnings, fmt.Sprintf("config file %s: tui.%s must be a positive integer (indent_step: 2 or 4)", path, key))
				} else if key == "tabstop" {
					code.Tabstop = int(number)
				} else {
					code.IndentStep = int(number)
				}
			case "literal_tab_alias":
				alias, ok := value.(string)
				if !ok || alias != "" && alias != "ctrl+q" && alias != "ctrl+b" {
					warnings = append(warnings, fmt.Sprintf("config file %s: tui.literal_tab_alias must be ctrl+q, ctrl+b or empty", path))
				} else {
					code.LiteralTabAlias = alias
				}
			default:
				warnings = append(warnings, fmt.Sprintf("config file %s: unknown setting tui.%s ignored", path, key))
			}
		}
	}
	return leader, ctrlW, code, warnings
}

func configDuration(path, key string, value interface{}) (*time.Duration, []string) {
	text, ok := value.(string)
	if !ok {
		return nil, []string{fmt.Sprintf("config file %s: %s must be a positive Go duration string", path, key)}
	}
	duration, err := time.ParseDuration(text)
	if err != nil || duration <= 0 {
		return nil, []string{fmt.Sprintf("config file %s: %s=%q must be a positive Go duration", path, key, text)}
	}
	return &duration, nil
}

func startupWarningSummary(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	suffix := ""
	if len(warnings) != 1 {
		suffix = "s"
	}
	return fmt.Sprintf("%d config warning%s — see :errors", len(warnings), suffix)
}

func parsePositiveDuration(flagName, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration (for example, 500ms or 2s); got %q", flagName, value)
	}
	return duration, nil
}
