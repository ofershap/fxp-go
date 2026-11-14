package fxp

import (
	"testing"
)

func j(t *testing.T, v any) string {
	t.Helper()
	b, err := MarshalJSON(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSmoke(t *testing.T) {
	p := New(DefaultOptions())
	v, err := p.Parse(`<root a="nice" checked><a>wow</a><b>2</b></root>`)
	if err != nil {
		t.Fatal(err)
	}
	got := j(t, v)
	want := `{"root":{"a":"wow","b":2}}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestAttrs(t *testing.T) {
	o := DefaultOptions()
	o.IgnoreAttributes = false
	v, err := New(o).Parse(`<root a="nice"><a>wow</a></root>`)
	if err != nil {
		t.Fatal(err)
	}
	got := j(t, v)
	want := `{"root":{"@_a":"nice","a":"wow"}}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestArray(t *testing.T) {
	v, _ := New(DefaultOptions()).Parse(`<r><i>1</i><i>2</i><i>3</i></r>`)
	got := j(t, v)
	want := `{"r":{"i":[1,2,3]}}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestEntities(t *testing.T) {
	v, _ := New(DefaultOptions()).Parse(`<r>a &amp; b &lt;x&gt;</r>`)
	got := j(t, v)
	want := `{"r":"a & b <x>"}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
