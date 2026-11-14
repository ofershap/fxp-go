package fxp

import "testing"

func FuzzParse(f *testing.F) {
	seeds := []string{
		`<r a="1"><b>x</b></r>`,
		`<?xml version="1.0"?><r>t</r>`,
		`<r><![CDATA[x]]><!--c--></r>`,
		`<!DOCTYPE r [<!ENTITY a "b">]><r>&a;</r>`,
		`<r><a>1</a></r></r><`,
		`<r a='1' b="2" c>`,
		`<x:y/>`,
		`<r>text &amp; &#65;</r>`,
		`<r><script>a < b</script></r>`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, mk := range []func() Options{
			DefaultOptions,
			func() Options { o := DefaultOptions(); o.IgnoreAttributes = false; o.PreserveOrder = true; return o },
			func() Options { o := DefaultOptions(); o.HTMLEntities = true; o.RemoveNSPrefix = true; return o },
			func() Options { o := DefaultOptions(); o.StopNodes = []string{"r.script"}; o.UnpairedTags = []string{"br"}; return o },
		} {
			p := New(mk())
			v, err := p.Parse(in)
			if err == nil {
				if _, berr := NewBuilder(DefaultBuildOptions()).Build(map[string]any{"out": v}); berr != nil {
					t.Fatalf("build after parse failed: %v", berr)
				}
			}
			_ = Validate(in)
		}
	})
}
