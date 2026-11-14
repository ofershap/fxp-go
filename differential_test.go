package fxp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type caseFile struct {
	Name    string         `json:"name"`
	XML     string         `json:"xml"`
	Options map[string]any `json:"options"`
}

type golden struct {
	Name  string          `json:"name"`
	Out   json.RawMessage `json:"out"`
	Error string          `json:"error"`
}

func optionsFromJSON(o map[string]any) Options {
	opts := DefaultOptions()
	for k, v := range o {
		switch k {
		case "ignoreAttributes":
			opts.IgnoreAttributes = v.(bool)
		case "allowBooleanAttributes":
			opts.AllowBooleanAttributes = v.(bool)
		case "parseAttributeValue":
			opts.ParseAttributeValue = v.(bool)
		case "parseTagValue":
			opts.ParseTagValue = v.(bool)
		case "parseTrueNumberOnly":
			opts.ParseTrueNumberOnly = v.(bool)
		case "trimValues":
			opts.TrimValues = v.(bool)
		case "processEntities":
			opts.ProcessEntities = v.(bool)
		case "htmlEntities":
			opts.HTMLEntities = v.(bool)
		case "removeNSPrefix":
			opts.RemoveNSPrefix = v.(bool)
		case "ignoreDeclaration":
			opts.IgnoreDeclaration = v.(bool)
		case "ignorePiTags":
			opts.IgnorePiTags = v.(bool)
		case "preserveOrder":
			opts.PreserveOrder = v.(bool)
		case "alwaysCreateTextNode":
			opts.AlwaysCreateTextNode = v.(bool)
		case "cdataPropName":
			opts.CdataPropName = v.(string)
		case "commentPropName":
			opts.CommentPropName = v.(string)
		case "textNodeName":
			opts.TextNodeName = v.(string)
		case "attributeNamePrefix":
			opts.AttributeNamePrefix = v.(string)
		case "attributesGroupName":
			opts.AttributesGroupName = v.(string)
		case "stopNodes":
			for _, s := range v.([]any) {
				opts.StopNodes = append(opts.StopNodes, s.(string))
			}
		case "unpairedTags":
			for _, s := range v.([]any) {
				opts.UnpairedTags = append(opts.UnpairedTags, s.(string))
			}
		case "maxNestedTags":
			opts.MaxNestedTags = int(v.(float64))
		default:
			panic("unknown option " + k)
		}
	}
	return opts
}

// normalizeJSON re-marshals a raw JSON message so map key order and
// number formatting compare byte-exact against Go output.
func normalizeJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var v any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	v = normalizeNumbers(v)
	b, err := MarshalJSON(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// normalizeNumbers converts json.Number to float64 when integral within
// float precision, keeping large values exact.
func normalizeNumbers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = normalizeNumbers(e)
		}
		return x
	case []any:
		for i, e := range x {
			x[i] = normalizeNumbers(e)
		}
		return x
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return f
	}
	return v
}

func TestDifferentialGoldens(t *testing.T) {
	caseBytes, err := os.ReadFile("testdata/gen/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	goldenBytes, err := os.ReadFile("testdata/gen/goldens.json")
	if err != nil {
		t.Fatal("goldens.json missing; run testdata/gen/gen.js")
	}
	var cases []caseFile
	var goldens []golden
	if err := json.Unmarshal(caseBytes, &cases); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(goldenBytes, &goldens); err != nil {
		t.Fatal(err)
	}
	gmap := map[string]golden{}
	for _, g := range goldens {
		gmap[g.Name] = g
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			g, ok := gmap[c.Name]
			if !ok {
				t.Fatal("no golden for case")
			}
			p := New(optionsFromJSON(c.Options))
			out, err := p.Parse(c.XML)
			if g.Error != "" {
				if err == nil {
					t.Fatalf("JS errored %q but Go succeeded: %v", g.Error, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("Go errored: %v (JS succeeded)", err)
			}
			got, merr := MarshalJSON(out)
			if merr != nil {
				t.Fatal(merr)
			}
			want := normalizeJSON(t, g.Out)
			if string(got) != want {
				t.Fatalf("\n got: %s\nwant: %s", got, want)
			}
		})
	}
}
