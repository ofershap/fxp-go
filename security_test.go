package fxp

import (
	"strings"
	"testing"
)

func TestEntityExpansionNotRecursive(t *testing.T) {
	// DOCTYPE entity values are inserted literally, never re-expanded:
	// a billion-laughs style chain stays small.
	xml := `<!DOCTYPE r [<!ENTITY a "&b;&b;&b;&b;&b;&b;&b;&b;&b;&b;"><!ENTITY b "xxxxxxxxxx">]><r>&a;</r>`
	v, err := New(DefaultOptions()).Parse(xml)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	// &a; expands to the literal text "&b;&b;..." (no recursion)
	if !strings.Contains(string(got), "&b;") {
		t.Fatalf("expected literal unexpanded entities, got %s", got)
	}
}

func TestDoctypeExternalIgnored(t *testing.T) {
	xml := `<!DOCTYPE r SYSTEM "http://evil.example/x.dtd"><r>1</r>`
	v, err := New(DefaultOptions()).Parse(xml)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if string(got) != `{"r":1}` {
		t.Fatalf("got %s", got)
	}
}

func TestDeepNestingUncapped(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 2000; i++ {
		b.WriteString("<n>")
	}
	b.WriteString("bottom")
	for i := 0; i < 2000; i++ {
		b.WriteString("</n>")
	}
	if _, err := New(DefaultOptions()).Parse(b.String()); err != nil {
		t.Fatal(err)
	}
}

func TestDeepNestingCapped(t *testing.T) {
	o := DefaultOptions()
	o.MaxNestedTags = 100
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("<n>")
	}
	if _, err := New(o).Parse(b.String()); err == nil {
		t.Fatal("want MaxNestedTags error")
	}
}

func TestCdataAndStopNodeContentNotDecoded(t *testing.T) {
	// CDATA is never entity-decoded
	v, err := New(DefaultOptions()).Parse(`<r><![CDATA[&amp;]]></r>`)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if !strings.Contains(string(got), "&amp;") {
		t.Fatalf("CDATA should stay raw, got %s", got)
	}
	// stop-node content is never entity-decoded
	o := DefaultOptions()
	o.StopNodes = []string{"r.s"}
	v2, err := New(o).Parse(`<r><s>&amp;</s></r>`)
	if err != nil {
		t.Fatal(err)
	}
	got2, _ := MarshalJSON(v2)
	if !strings.Contains(string(got2), "&amp;") {
		t.Fatalf("stop-node should stay raw, got %s", got2)
	}
}

func TestMalformedInputsErrorNotPanic(t *testing.T) {
	inputs := []string{
		"<", "<>", "<?", "<!", "<![CDATA[", "<!--", `<r a="`,
		"<r><a>", "</>", "< r>", "<r>&#xzz;</r>",
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on %q: %v", in, r)
				}
			}()
			_, _ = New(DefaultOptions()).Parse(in)
			_ = Validate(in)
		}()
	}
}

// TestDivergenceBareLtSpace pins the intentional divergence from the JS
// engine: "<" followed by a space outside stop nodes is a parse error here,
// while fast-xml-parser produces an empty-named element.
func TestDivergenceBareLtSpace(t *testing.T) {
	if _, err := New(DefaultOptions()).Parse(`<r><script>a < b</script></r>`); err == nil {
		t.Fatal("want parse error for bare '< ' input")
	}
}

func TestBuilderEscapesInvalidContent(t *testing.T) {
	b := NewBuilder(DefaultBuildOptions())
	xml, err := b.Build(map[string]any{"r": map[string]any{"@_a": `x<"&`, "#text": "<>&"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `a="x&lt;&quot;&amp;"`) || !strings.Contains(xml, "&lt;&gt;&amp;") {
		t.Fatalf("got %s", xml)
	}
	// re-parse the escaped output: values must come back exactly
	v, err := New(DefaultOptions()).Parse(xml)
	if err != nil {
		t.Fatal(err)
	}
	o := DefaultOptions()
	o.IgnoreAttributes = false
	v2, _ := New(o).Parse(xml)
	got, _ := MarshalJSON(v2)
	want := `{"r":{"#text":"<>&","@_a":"x<\"&"}}`
	if string(got) != want {
		t.Fatalf("roundtrip: got %s want %s", got, want)
	}
	_ = v
}

func TestBuilderDeepNesting(t *testing.T) {
	// 2000-deep nested maps build without stack issues
	m := map[string]any{"v": "bottom"}
	for i := 0; i < 2000; i++ {
		m = map[string]any{"n": m}
	}
	if _, err := NewBuilder(DefaultBuildOptions()).Build(map[string]any{"top": m}); err != nil {
		t.Fatal(err)
	}
}
