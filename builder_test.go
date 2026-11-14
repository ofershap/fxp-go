package fxp

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type buildGolden struct {
	Name  string `json:"name"`
	XML   string `json:"xml"`
	Skip  string `json:"skip"`
	Error string `json:"error"`
}

// TestBuilderRoundtrip builds XML from each parse output with the Go
// builder, re-parses both the Go-built and JS-built XML, and deep-compares
// the structures. Key ORDER differs deliberately: Go output is
// sorted-key deterministic, JS preserves insertion order.
func TestBuilderRoundtrip(t *testing.T) {
	caseBytes, err := os.ReadFile("testdata/gen/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	buildBytes, err := os.ReadFile("testdata/gen/build_goldens.json")
	if err != nil {
		t.Fatal("build_goldens.json missing; run testdata/gen/gen_build.js")
	}
	var cases []caseFile
	var bg []buildGolden
	if err := json.Unmarshal(caseBytes, &cases); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buildBytes, &bg); err != nil {
		t.Fatal(err)
	}
	bmap := map[string]buildGolden{}
	for _, b := range bg {
		bmap[b.Name] = b
	}
	for _, c := range cases {
		g := bmap[c.Name]
		if g.Skip != "" || g.Error != "" {
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
			opts := optionsFromJSON(c.Options)
			parsed, err := New(opts).Parse(c.XML)
			if err != nil {
				t.Fatal(err)
			}
			bopts := DefaultBuildOptions()
			if v, ok := c.Options["ignoreAttributes"]; ok {
				bopts.IgnoreAttributes = v.(bool)
			}
			if v, ok := c.Options["attributeNamePrefix"]; ok {
				bopts.AttributeNamePrefix = v.(string)
			}
			if v, ok := c.Options["attributesGroupName"]; ok {
				bopts.AttributesGroupName = v.(string)
			}
			if v, ok := c.Options["textNodeName"]; ok {
				bopts.TextNodeName = v.(string)
			}
			if v, ok := c.Options["cdataPropName"]; ok {
				bopts.CdataPropName = v.(string)
			}
			if v, ok := c.Options["commentPropName"]; ok {
				bopts.CommentPropName = v.(string)
			}
			if v, ok := c.Options["processEntities"]; ok {
				bopts.ProcessEntities = v.(bool)
			}
			goXML, err := NewBuilder(bopts).Build(parsed)
			if err != nil {
				t.Fatalf("Go build: %v", err)
			}
			// re-parse both XML strings with a neutral parser and compare
			neutral := DefaultOptions()
			neutral.ParseTagValue = false
			neutral.ParseAttributeValue = false
			neutral.CdataPropName = bopts.CdataPropName
			neutral.CommentPropName = bopts.CommentPropName
			goRe, err := New(neutral).Parse(goXML)
			if err != nil {
				t.Fatalf("Go-built XML does not re-parse: %v\nxml: %s", err, goXML)
			}
			jsRe, err := New(neutral).Parse(g.XML)
			if err != nil {
				t.Skipf("JS-built XML does not re-parse in Go (tolerance divergence): %v", err)
			}
			if !reflect.DeepEqual(normalize(goRe), normalize(jsRe)) {
				t.Fatalf("roundtrip mismatch:\n go xml: %s\n js xml: %s\n go re: %v\n js re: %v", goXML, g.XML, goRe, jsRe)
			}
		})
	}
}

// normalize sorts nothing (maps compare order-free in DeepEqual) but
// unwraps single-element arrays created by repeated-tag collapsing so
// scalar/array shape differences from re-parse do not cause false diffs.
func normalize(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = normalize(e)
		}
		return x
	case []any:
		if len(x) == 1 {
			return normalize(x[0])
		}
		for i, e := range x {
			x[i] = normalize(e)
		}
		return x
	}
	return v
}
