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

	if err := registry.Register("goto", []string{"g"}, func(gotModel *model, args []string) (tea.Model, tea.Cmd) {
		if gotModel != m {
			t.Error("handler received unexpected model")
		}
		gotArgs = args
		return gotModel, nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

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

func TestCommandRegistryResolvesUniquePrefixesAndExactAliases(t *testing.T) {
	registry := NewCommandRegistry()
	var called string
	if err := registry.Register("kanban", []string{"k"}, func(m *model, _ []string) (tea.Model, tea.Cmd) {
		called = "kanban"
		return m, nil
	}); err != nil {
		t.Fatalf("register kanban: %v", err)
	}
	if err := registry.Register("keep", nil, func(m *model, _ []string) (tea.Model, tea.Cmd) {
		called = "keep"
		return m, nil
	}); err != nil {
		t.Fatalf("register keep: %v", err)
	}

	if _, _, err := registry.Execute(&model{}, "kan"); err != nil || called != "kanban" {
		t.Fatalf("unique prefix = called:%q error:%v, want kanban", called, err)
	}
	if _, _, err := registry.Execute(&model{}, "k"); err != nil || called != "kanban" {
		t.Fatalf("exact alias = called:%q error:%v, want kanban", called, err)
	}
	if _, _, err := registry.Execute(&model{}, "ke"); err != nil || called != "keep" {
		t.Fatalf("unique prefix = called:%q error:%v, want keep", called, err)
	}
}

func TestCommandRegistryReportsStableAmbiguityAndCompletion(t *testing.T) {
	registry := NewCommandRegistry()
	handler := func(m *model, _ []string) (tea.Model, tea.Cmd) { return m, nil }
	if err := registry.Register("write", []string{"w"}, handler); err != nil {
		t.Fatalf("register write: %v", err)
	}
	if err := registry.Register("writequit", []string{"wq", "wq"}, handler); err != nil {
		t.Fatalf("register writequit: %v", err)
	}
	if err := registry.Register("writequitall", []string{"wqa"}, handler); err != nil {
		t.Fatalf("register writequitall: %v", err)
	}

	_, _, err := registry.Execute(&model{}, "wr")
	if err == nil || err.Error() != "ambiguous command: wr (matches: write, writequit, writequitall)" {
		t.Fatalf("ambiguity error = %v", err)
	}
	if completion, candidates := registry.Complete("w"); completion != "write" || !reflect.DeepEqual(candidates, []string{"write", "writequit", "writequitall"}) {
		t.Fatalf("completion = %q, candidates = %#v", completion, candidates)
	}
	if completion, candidates := registry.Complete("writequita"); completion != "writequitall" || candidates != nil {
		t.Fatalf("unique completion = %q, candidates = %#v", completion, candidates)
	}
}

func TestCommandRegistryRejectsSpellingCollisions(t *testing.T) {
	registry := NewCommandRegistry()
	handler := func(m *model, _ []string) (tea.Model, tea.Cmd) { return m, nil }
	if err := registry.Register("write", []string{"w"}, handler); err != nil {
		t.Fatalf("register write: %v", err)
	}
	if err := registry.Register("wipe", []string{"w"}, handler); err == nil || err.Error() != `command spelling "w" is already registered for "write"` {
		t.Fatalf("collision error = %v", err)
	}
	if _, _, err := registry.Execute(&model{}, "wipe"); err == nil || err.Error() != "unknown command: wipe" {
		t.Fatalf("failed registration changed registry: %v", err)
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
