package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func commentQuitModel() *model {
	m := testModel()
	m.store.(*fakeStore).taskComments = map[int64][]vimkunja.TaskComment{1: {}}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.commentsLoading = false
	m.openComments()
	return m
}

func enterModalCommand(m *model, command string) tea.Cmd {
	m.Update(tea.KeyPressMsg(tea.Key{Code: ':', Text: ":"}))
	m.Update(tea.KeyPressMsg(tea.Key{Text: command}))
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	return cmd
}

func TestEditableModalGlobalQuitConsistency(t *testing.T) {
	for _, modal := range []string{"detail", "comments", "labels", "search"} {
		for _, command := range []string{"qa", "quitall", " QA ", "qa!", "quitall!", "wqa", "writequitall", "writequita"} {
			t.Run(modal+"/"+command, func(t *testing.T) {
				m := testModel()
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
				switch modal {
				case "comments":
					m.openComments()
				case "labels":
					m.labelUI = &labelUI{kind: labelUITaskPicker}
				case "search":
					m.openSearch()
					m = pressKey(m, tea.Key{Code: tea.KeyEscape})
				}
				m.editor.draft.Title = "Changed"
				cmd := enterModalCommand(m, command)
				switch strings.ToLower(strings.TrimSpace(command)) {
				case "qa", "quitall":
					if cmd != nil || !strings.Contains(m.lastErr, "unsaved") {
						t.Fatalf("qa failed to protect Task draft: %q", m.lastErr)
					}
				case "qa!", "quitall!":
					requireQuit(t, cmd)
				default:
					if cmd == nil {
						t.Fatalf("missing save: %q", m.lastErr)
					}
					msg := cmd()
					if _, ok := msg.(taskSavedMsg); !ok {
						t.Fatalf("expected task save, got %T", msg)
					}
					_, quit := m.Update(msg)
					requireQuit(t, quit)
					if m.store.(*fakeStore).updatedDraft.Title != "Changed" {
						t.Fatal("Task draft was not saved")
					}
				}
			})
		}
	}
}

func TestModalGlobalDispatchLeavesCommentInsertTextUntouched(t *testing.T) {
	m := commentQuitModel()
	m.startCommentComposer()
	m = pressKey(m, tea.Key{Text: ":qa!"})
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || m.comments.composer.text != ":qa!\n" {
		t.Fatal("global dispatch intercepted literal Comment text")
	}
}

func TestModalGlobalDispatchDoesNotInterruptCommentSave(t *testing.T) {
	m := commentQuitModel()
	m.comments.saving = true
	m.comments.commandMode, m.comments.commandInput = true, "qa!"
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || !m.comments.commandMode || m.comments.commandInput != "qa!" {
		t.Fatal("global dispatch interrupted an in-flight save")
	}
}

func TestCommentWriteQuitAllRejectsEmptyDraftWithoutQuitIntent(t *testing.T) {
	m := commentQuitModel()
	m.startCommentComposer()
	m.comments.composer.mode = editorNormal
	if cmd := enterModalCommand(m, "wqa"); cmd != nil || m.quitAfterSave || m.comments.composer == nil || m.lastErr != "Comment text is required" {
		t.Fatalf("empty draft did not retain editor with validation error: %q", m.lastErr)
	}
}

func requireQuit(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected program quit, got no command")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestCommentsGlobalQuitCommands(t *testing.T) {
	for _, command := range []string{"qa", "quitall", "qa!", "quitall!"} {
		t.Run(command, func(t *testing.T) {
			m := commentQuitModel()
			requireQuit(t, enterModalCommand(m, command))
		})
	}
	for _, dirty := range []string{"comment", "task"} {
		t.Run(dirty, func(t *testing.T) {
			m := commentQuitModel()
			if dirty == "comment" {
				m.startCommentComposer()
				m.comments.composer.text = "Draft"
				m.comments.composer.mode = editorNormal
			} else {
				m.editor.draft.Title = "Changed"
			}
			if cmd := enterModalCommand(m, "qa"); cmd != nil || !strings.Contains(m.lastErr, "unsaved") {
				t.Fatalf("qa must protect unsaved %s, error=%q", dirty, m.lastErr)
			}
			requireQuit(t, enterModalCommand(m, "qa!"))
			if m.store.(*fakeStore).savedCommentDraft.Text != "" {
				t.Fatal("qa! saved a Comment")
			}
		})
	}
}

func TestCommentsWriteQuitAllSavesBeforeQuit(t *testing.T) {
	for _, command := range []string{"wqa", "writequitall"} {
		t.Run(command, func(t *testing.T) {
			m := commentQuitModel()
			m.editor.draft.Title = "Changed"
			m.startCommentComposer()
			m.comments.composer.text, m.comments.composer.mode = "Draft", editorNormal
			cmd := enterModalCommand(m, command)
			if cmd == nil {
				t.Fatalf("no save: %s", m.lastErr)
			}
			msg := cmd()
			if _, ok := msg.(taskCommentSavedMsg); !ok {
				t.Fatalf("must save Task and Comment atomically before quitting, got %T", msg)
			}
			_, quit := m.Update(msg)
			requireQuit(t, quit)
			store := m.store.(*fakeStore)
			if store.savedCommentDraft.Text != "Draft" || store.savedCommentTask.Title != "Changed" {
				t.Fatal("wqa omitted Task or Comment draft")
			}
		})
	}
}

func TestCommentsWriteQuitAllFailureKeepsDraftAndClearsQuitIntent(t *testing.T) {
	m := commentQuitModel()
	m.startCommentComposer()
	m.comments.composer.text, m.comments.composer.mode = "Draft", editorNormal
	m.store.(*fakeStore).commentErr = errors.New("write failed")
	cmd := enterModalCommand(m, "wqa")
	if cmd == nil {
		t.Fatalf("no save: %s", m.lastErr)
	}
	_, next := m.Update(cmd())
	if next != nil || m.quitAfterSave || m.comments.composer == nil || m.comments.composer.text != "Draft" || m.lastErr != "write failed" {
		t.Fatal("failed save lost draft or retained quit intent")
	}
	m.store.(*fakeStore).commentErr = nil
	cmd = enterModalCommand(m, "w")
	if cmd == nil {
		t.Fatal("retry did not save")
	}
	_, next = m.Update(cmd())
	if next != nil {
		if _, quits := next().(tea.QuitMsg); quits {
			t.Fatal("ordinary retry quit after failed wqa")
		}
	}
}
