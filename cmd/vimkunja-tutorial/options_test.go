package main

import (
	"io"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/tutorial"
)

func TestTutorialModeFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mode      tutorial.Mode
		menu      bool
		wantError bool
	}{
		{name: "no flags opens menu", menu: true},
		{name: "timeouts alone keep menu", args: []string{"--leader-timeout", "1s", "--ctrl-w-timeout", "2s"}, menu: true},
		{name: "tab alias alone keeps menu", args: []string{"--literal-tab-alias", "ctrl+b"}, menu: true},
		{name: "guided basics", args: []string{"--guided"}},
		{name: "practice basics", args: []string{"--practice"}, mode: tutorial.Mode{Practice: true}},
		{name: "guided markup", args: []string{"--markup"}, mode: tutorial.Mode{Markup: true}},
		{name: "practice markup", args: []string{"--markup", "--practice"}, mode: tutorial.Mode{Markup: true, Practice: true}},
		{name: "explicit guided markup", args: []string{"--markup", "--guided"}, mode: tutorial.Mode{Markup: true}},
		{name: "conflicting guidance", args: []string{"--guided", "--practice"}, wantError: true},
		{name: "invalid alias", args: []string{"--literal-tab-alias", "wrong"}, wantError: true},
		{name: "unexpected argument", args: []string{"typo"}, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptions(tt.args, io.Discard)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError %v", err, tt.wantError)
			}
			if err == nil && (got.mode != tt.mode || got.menu != tt.menu) {
				t.Fatalf("mode = %+v, menu %v; want %+v, %v", got.mode, got.menu, tt.mode, tt.menu)
			}
		})
	}
}

func TestTutorialFlagsPreserveExplicitTimeoutOverrides(t *testing.T) {
	got, err := parseOptions([]string{"--leader-timeout", "1s", "--ctrl-w-timeout", "750ms", "--literal-tab-alias", "ctrl+b"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !got.overrides.LeaderTimeoutSet || got.overrides.LeaderTimeout != "1s" || !got.overrides.CtrlWTimeoutSet || got.overrides.CtrlWTimeout != "750ms" || got.literalTabAlias != "ctrl+b" {
		t.Fatalf("options = %+v", got)
	}
	defaults, err := parseOptions(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.overrides.LeaderTimeoutSet || defaults.overrides.CtrlWTimeoutSet {
		t.Fatal("absent flags must not override user configuration")
	}
}
