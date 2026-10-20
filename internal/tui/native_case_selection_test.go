package tui

import (
	"reflect"
	"testing"
)

func TestNativeVisualToggleCase(t *testing.T) {
	for _, keys := range [][]string{{"v", "i", "w"}, {"v", "4", "l"}, {"4", "l", "v", "4", "h"}} {
		const source = `<p>Alpha batea und so weiter</p>`
		m := nativeKeys(nativeEditingModel(source), keys...)
		beforeSelection := m.editor.nativeDraft.Selection()
		beforeCursor := m.editor.nativeDraft.Cursor()
		m = nativeKeys(m, "~")
		if m.lastErr != "" || m.editor.draft.Description != `<p>aLPHA batea und so weiter</p>` {
			t.Fatalf("%v: %s: %s", keys, m.lastErr, m.editor.draft.Description)
		}
		if !reflect.DeepEqual(m.editor.nativeDraft.Selection(), beforeSelection) || m.editor.nativeDraft.Cursor() != beforeCursor || !m.editor.dirty() {
			t.Fatal("case change should retain selection/cursor and dirty the draft")
		}
		m = nativeKeys(m, "u")
		if m.editor.draft.Description != source {
			t.Fatal("Undo did not restore selection text")
		}
		m = nativeKeys(m, "ctrl+r")
		if m.editor.draft.Description != `<p>aLPHA batea und so weiter</p>` {
			t.Fatal("Redo did not restore case change")
		}
	}
}

func TestNativeVisualToggleCasePreservesStructure(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		keys               []string
	}{
		{"plain", `Alpha batea`, `<p>aLPHA batea</p>`, []string{"v", "i", "w"}},
		{"unicode", `<p>Ⱥlpha Äß é 👩‍💻</p>`, `<p>ⱥLPHA äß É 👩‍💻</p>`, []string{"V"}},
		{"inline", `<p><strong>Alpha</strong> <a href="/Alpha" target="_blank">Beta</a></p>`, `<p><strong>aLPHA</strong> <a href="/Alpha" target="_blank">bETA</a></p>`, []string{"V"}},
		{"blocks", `<p>Alpha</p><p>Beta</p><p>Keep</p>`, `<p>aLPHA</p><p>bETA</p><p>Keep</p>`, []string{"V", "j"}},
		{"protected", `<p>Alpha<img src='Alpha' alt='Beta'>Beta</p>`, `<p>aLPHA<img src='Alpha' alt='Beta'>bETA</p>`, []string{"V"}},
		{"code", "<pre><code>Alpha\nBeta\nKeep</code></pre>", "<pre><code>aLPHA\nbETA\nKeep</code></pre>", []string{"V", "j"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeKeys(nativeEditingModel(tc.source), tc.keys...)
			m = nativeKeys(m, "~")
			if m.lastErr != "" || m.editor.draft.Description != tc.want {
				t.Fatalf("%s: %s; want %s", m.lastErr, m.editor.draft.Description, tc.want)
			}
			if m.editor.nativeDraft.Selection() == nil {
				t.Fatal("lost Visual selection")
			}
			if _, err := m.editor.nativeDraft.PrepareSave(); err != nil {
				t.Fatal(err)
			}
			m = nativeKeys(m, "u")
			if m.editor.draft.Description != tc.source {
				t.Fatal("Undo changed source", m.editor.draft.Description)
			}
		})
	}
}

func TestNativeVisualToggleCaseRemapsUnicodeSelection(t *testing.T) {
	for _, keys := range [][]string{{"v", "i", "w"}, {"4", "l", "v", "4", "h"}} {
		m := nativeKeys(nativeEditingModel(`<p>Ⱥlpha Keep</p>`), keys...)
		m = nativeKeys(m, "~")
		if m.lastErr != "" || m.editor.nativeDraft.Copy() != "ⱥLPHA" {
			t.Fatalf("selection lost after UTF-8 width change: %q %s", m.editor.nativeDraft.Copy(), m.lastErr)
		}
		m = nativeKeys(m, "~")
		if m.lastErr != "" || m.editor.nativeDraft.Copy() != "Ⱥlpha" || m.editor.draft.Description != `<p>Ⱥlpha Keep</p>` {
			t.Fatal("second toggle did not reuse the selected text", m.editor.draft.Description, m.lastErr)
		}
		m = nativeKeys(m, "esc")
		if m.editor.nativeDraft.Selection() != nil {
			t.Fatal("Esc retained selection")
		}
	}
}
