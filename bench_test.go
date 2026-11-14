package fxp

import (
	x "encoding/xml"
	"testing"
)

var benchXML = `<catalog count="100">
  <product id="p1"><name>Widget</name><price currency="USD">19.99</price><tags><tag>a</tag><tag>b</tag></tags></product>
  <product id="p2"><name>Gadget</name><price currency="EUR">9.99</price><desc>multi <b>mixed</b> text</desc></product>
  <product id="p3"><name>Thing</name><price>0x10</price></product>
</catalog>`

func BenchmarkParse(b *testing.B) {
	p := New(DefaultOptions())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := p.Parse(benchXML); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlibUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var v any
		_ = x.Unmarshal([]byte(benchXML), &v)
	}
}
