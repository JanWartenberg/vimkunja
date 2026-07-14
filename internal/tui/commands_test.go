package tui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCommandRegistryExecutesNameAndAlias(t *testing.T) {
	registry := NewCommandRegistry()
	m := &model{}
	var gotArgs []string

	registry.Register("goto", []string{"g"}, func(gotModel *model, args []string) (tea.Model, tea.Cmd) {
		if gotModel != m {
			t.Error("handler received unexpected model")
		}
		gotArgs = args
		return gotModel, nil
	})

	updated, _, err := registry.Execute(m, ":GOTO  project 42")
	if err != nil {
		t.Fatalf("execute canonical name: %v", err)
	}
	if updated != m {
		t.Fatal("handler returned unexpected model")
	}
	if want := []string{"project", "42"}; !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("canonical args = %#v, want %#v", gotArgs, want)
	}

	gotArgs = nil
	_, _, err = registry.Execute(m, "g target")
	if err != nil {
		t.Fatalf("execute alias: %v", err)
	}
	if want := []string{"target"}; !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("alias args = %#v, want %#v", gotArgs, want)
	}
}

func TestCommandRegistryRejectsEmptyCommand(t *testing.T) {
	_, _, err := NewCommandRegistry().Execute(&model{}, " :   ")
	if err == nil || err.Error() != "empty command" {
		t.Fatalf("error = %v, want empty command", err)
	}
}

func TestCommandRegistryRejectsUnknownCommand(t *testing.T) {
	_, _, err := NewCommandRegistry().Execute(&model{}, ":missing")
	if err == nil || err.Error() != "unknown command: missing" {
		t.Fatalf("error = %v, want unknown-command error", err)
	}
}
