# TODO: UX Improvements & Design Decisions

**Status:** Open questions for discussion  
**Priority:** Mixed (some P1, some P3)

---

## 🔥 P2: Task Description Rendering (NEEDS DISCUSSION)

**Issue:** Task descriptions in Vikunja are stored as **HTML** (rich text from Tiptap editor in web UI).

**Current state:**
- Detail pane shows raw description field
- May contain HTML tags like `<p>`, `<strong>`, `<ul>`, `<li>`, etc.
- Not rendering in TUI (just shows raw string)

**Problem:**
```
Description: "<p>This is <strong>bold</strong> and this is <em>italic</em>.</p>"
```

Displays as literal HTML in TUI → unreadable!

**Options:**

### A) Strip HTML Tags (Simple)
**Pros:**
- Easy to implement (regex or `html.UnescapeString`)
- Always works
- No external dependencies

**Cons:**
- Loses formatting information
- Lists become flat text
- Links lose context

**Implementation:**
```go
import "html"
import "regexp"

func stripHTML(s string) string {
    re := regexp.MustCompile(`<[^>]*>`)
    stripped := re.ReplaceAllString(s, "")
    return html.UnescapeString(stripped)
}
```

---

### B) Convert HTML → Markdown (Medium)
**Pros:**
- Preserves some formatting (bold, italic, lists)
- Markdown is TUI-friendly
- Can render with lipgloss styles

**Cons:**
- Needs HTML→Markdown library (e.g., `github.com/JohannesKaufmann/html-to-markdown`)
- Dependency overhead
- Not perfect conversion

**Implementation:**
```go
import "github.com/JohannesKaufmann/html-to-markdown"

func htmlToMarkdown(s string) string {
    converter := md.NewConverter("", true, nil)
    markdown, _ := converter.ConvertString(s)
    return markdown
}
```

---

### C) Custom HTML Renderer for TUI (Complex)
**Pros:**
- Full control over rendering
- Can map HTML tags to lipgloss styles
- Best UX

**Cons:**
- Significant implementation effort
- Need to handle all HTML edge cases
- Complex state machine

**Implementation sketch:**
```go
// Parse HTML, render styled blocks
<p>      → normal text
<strong> → lipgloss.Bold()
<em>     → lipgloss.Italic()
<ul><li> → "• " prefix
<a href> → underline + show URL in parens
```

---

### D) Minimal: Just Remove Tags + Show Plain Text (Quick Win)
**Pros:**
- Zero dependencies
- Can ship today
- "Good enough" for MVP

**Cons:**
- Loses ALL formatting
- Same as option A

**Implementation:**
```go
// In internal/tui/panes.go or similar:
func renderDescription(html string) string {
    // Quick regex strip
    re := regexp.MustCompile(`<[^>]*>`)
    plain := re.ReplaceAllString(html, "")
    // Unescape entities
    plain = strings.ReplaceAll(plain, "&nbsp;", " ")
    plain = strings.ReplaceAll(plain, "&lt;", "<")
    plain = strings.ReplaceAll(plain, "&gt;", ">")
    plain = strings.ReplaceAll(plain, "&amp;", "&")
    return plain
}
```

---

## **Decision Needed:**

**Question for User:**
1. Which option do you prefer? (A/B/C/D)
2. Is preserving formatting (bold, lists) important?
3. Are you OK with adding a dependency for HTML→Markdown conversion?

**Recommendation (Pi):**
- **MVP (Phase 1-7):** Option D (strip tags, zero deps)
- **Post-MVP (Phase 8+):** Option B (HTML→Markdown with lipgloss styling)

**Rationale:**
- Option D ships fast, unblocks UX testing
- Option B can be added later without breaking changes
- Option C is overkill for TUI use case

---

## 🔄 Other UX TODOs (Lower Priority)

### P3: Long Title Truncation
**Issue:** Task titles can be very long, may overflow pane width.

**Options:**
- Truncate with `...` (current behavior in many TUIs)
- Word-wrap (lipgloss `Width()` + wrapping)
- Scroll horizontally on focus (complex)

**Recommendation:** Word-wrap with max 3 lines, then truncate.

---

### P3: Date Formatting
**Issue:** `due_date` is ISO8601 string (`2026-07-10T14:30:00Z`).

**Current:** Shows raw string (long, cluttered)

**Options:**
- Relative: "in 2 days", "tomorrow", "overdue by 3 days"
- Short: "Jul 10", "2026-07-10"
- Conditional: Relative if <7 days, else absolute

**Recommendation:** Relative for <7 days, "YYYY-MM-DD" otherwise.

---

### P4: Empty Description Placeholder
**Issue:** When `description == ""`, detail pane shows empty field.

**Options:**
- Show "—" or "(no description)"
- Hide field entirely
- Gray italic "(add description)"

**Recommendation:** Gray italic "(no description set)"

---

### P4: Priority Display Consistency
**Issue:** Priority shown as number (1=urgent, 5=low) or color?

**Current:** Color-coded in task list (red=urgent, green=low)

**Detail pane:** Should we show "Priority: 🔴 Urgent" or just color bar?

**Recommendation:** Color + label (e.g., "🔴 Urgent", "🟢 Low")

---

### P5: Task Index/Identifier
**Issue:** Some tasks have `identifier` (e.g., "PROJ-123"), others just `#<index>`.

**Current:** Not shown in TUI

**Options:**
- Show in task list: `[PROJ-123] Task title`
- Show in detail pane: `Identifier: PROJ-123`
- Don't show (just use ID internally)

**Recommendation:** Show in detail pane only (keeps list clean).

---

## Next Steps

1. **User decides on Description rendering** (A/B/C/D)
2. Implement chosen approach in Phase 2-5 or Phase 8
3. Add to `NEXT_STEPS.md` or defer to "Polish" phase

---

**Related Spec:** SPEC_FUNDAMENT.md (Detail-pane rendering)  
**Related Code:** `internal/tui/panes.go` (renderDetailPane)

**Last Updated:** 2026-07-10  
**Status:** Awaiting user input on Description rendering
