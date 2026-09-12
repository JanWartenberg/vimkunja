package tutorial

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type recordingModel struct {
	messages []tea.Msg
}

func (m *recordingModel) Init() tea.Cmd { return nil }

func (m *recordingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, msg)
	return m, nil
}

func (m *recordingModel) View() tea.View {
	view := tea.NewView("APPLICATION")
	view.AltScreen = true
	return view
}

func TestTutorialConsumesWelcomeAndForwardsOnlyExpectedLessonKeys(t *testing.T) {
	app := &recordingModel{}
	model := NewModel(app).(*Model)

	model.Update(keyPress("enter"))
	if len(app.messages) != 0 {
		t.Fatalf("welcome Enter reached app: %#v", app.messages)
	}
	if model.step != 1 {
		t.Fatalf("step = %d, want first interactive lesson", model.step)
	}

	model.Update(keyPress("x"))
	if len(app.messages) != 0 {
		t.Fatalf("unexpected key reached app: %#v", app.messages)
	}
	if !strings.Contains(model.feedback, "ignored") {
		t.Fatalf("feedback = %q, want ignored-key explanation", model.feedback)
	}

	model.Update(keyPress("j"))
	if len(app.messages) != 1 {
		t.Fatalf("forwarded message count = %d, want 1", len(app.messages))
	}
	if model.key != 1 {
		t.Fatalf("lesson key progress = %d, want 1", model.key)
	}
}

func TestTutorialSkipsInvisibleTitleCursorAndReachesWorkflowStatus(t *testing.T) {
	steps := implementedTutorialSteps()
	for _, step := range steps {
		if step.title == "Title Motions" {
			t.Fatal("tutorial must not teach the invisible Normal-mode title cursor")
		}
	}

	var workflow tutorialStep
	for _, step := range steps {
		if step.title == "Workflow Status" {
			workflow = step
			break
		}
	}
	if len(workflow.keys) != 2 || workflow.keys[0].value != "j" || workflow.keys[1].value != "j" {
		t.Fatalf("workflow status keys = %#v, want j, j", workflow.keys)
	}
}

func TestTutorialResizesRealApplicationAboveInstructionStrip(t *testing.T) {
	app := &recordingModel{}
	model := NewModel(app).(*Model)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if len(app.messages) != 1 {
		t.Fatalf("messages = %d, want resize", len(app.messages))
	}
	size, ok := app.messages[0].(tea.WindowSizeMsg)
	if !ok {
		t.Fatalf("message = %T, want WindowSizeMsg", app.messages[0])
	}
	if size.Width != 100 || size.Height != 30-tutorialPanelHeight {
		t.Fatalf("app size = %dx%d", size.Width, size.Height)
	}
	view := model.View()
	if !view.AltScreen || !strings.Contains(view.Content, "APPLICATION") || !strings.Contains(view.Content, "VIMKUNJA TUTORIAL") {
		t.Fatalf("composed view = %#v", view)
	}
}

func keyPress(value string) tea.KeyPressMsg {
	switch value {
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	default:
		runes := []rune(value)
		return tea.KeyPressMsg(tea.Key{Code: runes[0], Text: value})
	}
}
