package fxp

import (
	"strings"
	"testing"
)

func TestIsArrayCallback(t *testing.T) {
	o := DefaultOptions()
	o.IsArray = func(tagName, jPath string, isLeaf, isAttr bool) bool {
		return tagName == "i"
	}
	v, err := New(o).Parse(`<r><i>1</i></r>`)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if string(got) != `{"r":{"i":[1]}}` {
		t.Fatalf("got %s", got)
	}
}

func TestUpdateTagRenameAndDrop(t *testing.T) {
	o := DefaultOptions()
	o.UpdateTag = func(tagName, jPath, attrs string) (string, bool) {
		if tagName == "drop" {
			return "", false
		}
		if tagName == "old" {
			return "new", true
		}
		return tagName, true
	}
	v, err := New(o).Parse(`<r><old>1</old><drop>2</drop></r>`)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if string(got) != `{"r":{"new":1}}` {
		t.Fatalf("got %s", got)
	}
}

func TestTransforms(t *testing.T) {
	o := DefaultOptions()
	o.IgnoreAttributes = false
	o.TransformTagName = strings.ToUpper
	o.TransformAttributeName = strings.ToUpper
	v, err := New(o).Parse(`<r a="1"><b>x</b></r>`)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if string(got) != `{"R":{"@_A":"1","B":"x"}}` {
		t.Fatalf("got %s", got)
	}
}

func TestValueProcessors(t *testing.T) {
	o := DefaultOptions()
	o.IgnoreAttributes = false
	o.ParseTagValue = false
	o.TagValueProcessor = func(name, val string) string { return "T:" + val }
	o.AttributeValueProcessor = func(name, val string) string { return "A:" + val }
	v, err := New(o).Parse(`<r a="1">x</r>`)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := MarshalJSON(v)
	if string(got) != `{"r":{"#text":"T:x","@_a":"A:1"}}` {
		t.Fatalf("got %s", got)
	}
}

func TestValidate(t *testing.T) {
	valid := []string{
		`<r><a>x</a></r>`,
		`<?xml version="1.0"?><r a="1" b>x</r>`,
		`<r><![CDATA[<x>]]><!-- c --></r>`,
		`<r/>`,
		`<x:y a:b="c"/>`,
	}
	for _, x := range valid {
		if err := Validate(x); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", x, err)
		}
	}
	invalid := []string{
		`<r><a></r>`,
		`<r>`,
		`</r>`,
		`<r a=1/>`,
		`<r a="1/>`,
		`<1r/>`,
		`<r></q>`,
		`<r><!-- x </r>`,
	}
	for _, x := range invalid {
		if err := Validate(x); err == nil {
			t.Errorf("Validate(%q) = nil, want error", x)
		}
	}
}

func TestBuilderBasics(t *testing.T) {
	b := NewBuilder(DefaultBuildOptions())
	xml, err := b.Build(map[string]any{"r": map[string]any{"@_a": "1", "i": []any{1.0, 2.0}, "#text": "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `<r a="1">`) || !strings.Contains(xml, "<i>1</i>") || !strings.Contains(xml, "<i>2</i>") {
		t.Fatalf("got %s", xml)
	}
	if err := Validate(xml); err != nil {
		t.Fatalf("built XML invalid: %v", err)
	}
}

func TestBuilderEscapes(t *testing.T) {
	b := NewBuilder(DefaultBuildOptions())
	xml, _ := b.Build(map[string]any{"r": map[string]any{"@_a": `x<"&`, "#text": "<>&"}})
	if !strings.Contains(xml, `a="x&lt;&quot;&amp;"`) || !strings.Contains(xml, "&lt;&gt;&amp;") {
		t.Fatalf("got %s", xml)
	}
}

func TestMaxNestedTags(t *testing.T) {
	o := DefaultOptions()
	o.MaxNestedTags = 2
	if _, err := New(o).Parse(`<a><b><c>x</c></b></a>`); err == nil {
		t.Fatal("want depth error")
	}
	if _, err := New(o).Parse(`<a><b>x</b></a>`); err != nil {
		t.Fatal(err)
	}
}


