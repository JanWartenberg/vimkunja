package verification

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"golang.org/x/net/html"
)

// These tests check specification artifacts and dependency assumptions.
// Native document/editor execution has separate package suites.
func TestCorpusArtifactConsistency(t *testing.T) {
	data, err := os.ReadFile("../corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID             string `json:"id"`
			Classification string `json:"classification"`
			Source         string `json:"source_html"`
			Serialization  struct {
				HTML       *string  `json:"html"`
				Opaque     []string `json:"opaque_bytes"`
				Assertions []string `json:"assertions"`
			} `json:"serialization"`
			Edits []struct {
				Action     string   `json:"action"`
				HTML       *string  `json:"expected_html"`
				Assertions []string `json:"assertions"`
			} `json:"edits"`
			Unresolved []string `json:"unresolved"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile("../corpus.md")
	if err != nil {
		t.Fatal(err)
	}
	currentIndex := strings.SplitN(string(index), "## Historical review log", 2)[0]
	seen := map[string]bool{}
	for _, f := range corpus.Fixtures {
		t.Run(f.ID, func(t *testing.T) {
			if f.ID == "" || seen[f.ID] {
				t.Fatal("missing or duplicate fixture ID")
			}
			seen[f.ID] = true
			if !strings.Contains(currentIndex, "`"+f.ID+"`") {
				t.Fatal("fixture missing from current traceability matrix")
			}
			switch f.Classification {
			case "Native", "mixed Native/Opaque", "Blocked":
			default:
				t.Fatalf("unknown classification %q", f.Classification)
			}
			if f.Serialization.HTML == nil && len(f.Serialization.Assertions) == 0 {
				t.Fatal("null ordinary-save output needs explicit assertions")
			}
			if f.Classification == "Blocked" && f.Serialization.HTML != nil {
				t.Fatal("Blocked fixture cannot claim an ordinary successful save")
			}
			// Advance independently through source and output: repeated identical
			// atoms must occur in order and with their full multiplicity.
			for _, text := range []*string{&f.Source, f.Serialization.HTML} {
				if text == nil {
					continue
				}
				offset := 0
				for _, atom := range f.Serialization.Opaque {
					if atom == "" {
						t.Fatal("empty opaque source slice")
					}
					index := strings.Index((*text)[offset:], atom)
					if index < 0 {
						t.Fatalf("missing or reordered opaque slice %q", atom)
					}
					offset += index + len(atom)
				}
			}
			for _, edit := range f.Edits {
				if edit.Action == "" || (edit.HTML == nil && len(edit.Assertions) == 0) {
					t.Fatalf("edit requires action and justified output/assertions: %#v", edit)
				}
			}
			z := html.NewTokenizer(strings.NewReader(f.Source))
			var raw strings.Builder
			for {
				kind := z.Next()
				raw.WriteString(string(z.Raw()))
				if kind == html.ErrorToken {
					if z.Err() != io.EOF {
						t.Fatal(z.Err())
					}
					break
				}
			}
			if raw.String() != f.Source {
				t.Fatal("tokenizer Raw copies lost original source bytes")
			}
		})
	}
	t.Logf("Validated %d independently authored fixture records; no Native execution claim", len(corpus.Fixtures))
}

func TestTokenizerRawBytesAndDecodedCode(t *testing.T) {
	const source = "<pre><code>\nA\r\n\tB &amp; &lt;tag&gt;\r\n</code></pre><custom-node DATA-X='one' data-x=\"two\">é</custom-node>"
	z := html.NewTokenizer(strings.NewReader(source))
	var raw strings.Builder
	var codeText string
	inCode := false
	for {
		tokenType := z.Next()
		// Raw must be copied before Token/Text calls, which can mutate the
		// tokenizer buffer while normalizing or unescaping it.
		raw.WriteString(string(z.Raw()))
		if tokenType == html.ErrorToken {
			if z.Err() != io.EOF {
				t.Fatal(z.Err())
			}
			break
		}
		token := z.Token()
		if tokenType == html.StartTagToken && token.Data == "code" {
			inCode = true
		} else if tokenType == html.EndTagToken && token.Data == "code" {
			inCode = false
		} else if tokenType == html.TextToken && inCode {
			codeText += token.Data
		}
	}
	if raw.String() != source {
		t.Fatal("Raw token copies did not reproduce original UTF-8 source")
	}
	if codeText != "\nA\n\tB & <tag>\n" {
		t.Fatalf("decoded Code dependency behavior changed: %q", codeText)
	}
	t.Log("Raw copies retain CRLF; Token/Text normalizes CRLF to LF. Native Code must decode original text slices, not use Token.Data unchanged.")
}

func TestTreeParserRepairsMalformedHTMLWithoutError(t *testing.T) {
	for _, source := range []string{"<p>Visible <strong>unclosed", "<p>A<div>B</div>C</p>"} {
		// A nil context uses the parser's normal fragment behavior. Successful
		// parsing alone must never be taken as strict structural validation.
		nodes, err := html.ParseFragment(strings.NewReader(source), nil)
		if err != nil || len(nodes) == 0 {
			t.Fatalf("dependency repair probe: %q => %v", source, err)
		}
	}
	t.Log("Malformed/unexpectedly repaired HTML returns no parse error; explicit boundary validation is required.")
}

func TestTerminalDecoderCharacterBindings(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"ü", "ü"}, {"Ü", "Ü"}, {"]", "]"}, {"[", "["}, {"ö", "ö"}, {"ä", "ä"},
		{"`", "`"}, {"\x16", "ctrl+v"}, {"\x11", "ctrl+q"}, {"\x02", "ctrl+b"}, {"\t", "tab"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			var decoder uv.EventDecoder
			n, event := decoder.Decode([]byte(tc.input))
			key, ok := event.(uv.KeyPressEvent)
			if n != len(tc.input) || !ok || key.String() != tc.want {
				t.Fatalf("decode %q = %d/%#v; want %s", tc.input, n, event, tc.want)
			}
		})
	}
	var decoder uv.EventDecoder
	buffer := []byte("````\r")
	var keys []string
	for len(buffer) > 0 {
		n, event := decoder.Decode(buffer)
		if n <= 0 {
			t.Fatal("decoder made no progress")
		}
		key, ok := event.(uv.KeyPressEvent)
		if !ok {
			t.Fatalf("unexpected event %#v", event)
		}
		keys = append(keys, key.String())
		buffer = buffer[n:]
	}
	if len(keys) != 5 || strings.Join(keys[:4], "") != "````" || keys[4] != "enter" {
		t.Fatalf("lost fourth backtick: %#v", keys)
	}
	t.Log("Synthetic byte sequences only: no Windows/AltGr/dead-key hardware capture or native editor dispatch evidence.")
}
