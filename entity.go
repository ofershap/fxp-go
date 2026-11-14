package fxp

import (
	"strconv"
	"strings"
)

var xmlEntities = map[string]string{
	"amp": "&", "lt": "<", "gt": ">", "quot": "\"", "apos": "'",
}

// Common HTML named entities (subset covering the entities5.json entries
// most seen in real feeds; numeric refs are fully supported).
var htmlEntities = map[string]string{
	"nbsp": "\u00a0", "copy": "\u00a9", "reg": "\u00ae", "trade": "\u2122",
	"hellip": "\u2026", "mdash": "\u2014", "ndash": "\u2013",
	"lsquo": "\u2018", "rsquo": "\u2019", "ldquo": "\u201c", "rdquo": "\u201d",
	"laquo": "\u00ab", "raquo": "\u00bb", "times": "\u00d7", "divide": "\u00f7",
	"deg": "\u00b0", "plusmn": "\u00b1", "micro": "\u00b5", "para": "\u00b6",
	"sect": "\u00a7", "middot": "\u00b7", "bull": "\u2022", "dagger": "\u2020",
	"Dagger": "\u2021", "permil": "\u2030", "prime": "\u2032", "Prime": "\u2033",
	"euro": "\u20ac", "pound": "\u00a3", "yen": "\u00a5", "cent": "\u00a2",
	"curren": "\u00a4", "brvbar": "\u00a6", "uml": "\u00a8", "ordf": "\u00aa",
	"ordm": "\u00ba", "iexcl": "\u00a1", "iquest": "\u00bf", "szlig": "\u00df",
	"agrave": "\u00e0", "aacute": "\u00e1", "acirc": "\u00e2", "atilde": "\u00e3",
	"auml": "\u00e4", "aring": "\u00e5", "aelig": "\u00e6", "ccedil": "\u00e7",
	"egrave": "\u00e8", "eacute": "\u00e9", "ecirc": "\u00ea", "euml": "\u00eb",
	"igrave": "\u00ec", "iacute": "\u00ed", "icirc": "\u00ee", "iuml": "\u00ef",
	"ntilde": "\u00f1", "ograve": "\u00f2", "oacute": "\u00f3", "ocirc": "\u00f4",
	"otilde": "\u00f5", "ouml": "\u00f6", "oslash": "\u00f8", "ugrave": "\u00f9",
	"uacute": "\u00fa", "ucirc": "\u00fb", "uuml": "\u00fc", "yacute": "\u00fd",
	"yuml": "\u00ff", "Agrave": "\u00c0", "Aacute": "\u00c1", "Acirc": "\u00c2",
	"Atilde": "\u00c3", "Auml": "\u00c4", "Aring": "\u00c5", "AElig": "\u00c6",
	"Ccedil": "\u00c7", "Egrave": "\u00c8", "Eacute": "\u00c9", "Ecirc": "\u00ca",
	"Euml": "\u00cb", "Igrave": "\u00cc", "Iacute": "\u00cd", "Icirc": "\u00ce",
	"Iuml": "\u00cf", "Ntilde": "\u00d1", "Ograve": "\u00d2", "Oacute": "\u00d3",
	"Ocirc": "\u00d4", "Otilde": "\u00d5", "Ouml": "\u00d6", "Oslash": "\u00d8",
	"Ugrave": "\u00d9", "Uacute": "\u00da", "Ucirc": "\u00db", "Uuml": "\u00dc",
	"Yacute": "\u00dd", "alpha": "\u03b1", "beta": "\u03b2", "gamma": "\u03b3",
	"delta": "\u03b4", "epsilon": "\u03b5", "theta": "\u03b8", "lambda": "\u03bb",
	"mu": "\u03bc", "pi": "\u03c0", "sigma": "\u03c3", "phi": "\u03c6",
	"omega": "\u03c9", "Alpha": "\u0391", "Beta": "\u0392", "Gamma": "\u0393",
	"Delta": "\u0394", "Theta": "\u0398", "Lambda": "\u039b", "Pi": "\u03a0",
	"Sigma": "\u03a3", "Phi": "\u03a6", "Omega": "\u03a9",
	"larr": "\u2190", "uarr": "\u2191", "rarr": "\u2192", "darr": "\u2193",
	"harr": "\u2194", "spades": "\u2660", "clubs": "\u2663",
	"hearts": "\u2665", "diams": "\u2666",
}

// decodeEntities replaces entity references. doctype holds DOCTYPE-defined
// entities; html enables the HTML named set. Unknown entities are kept as-is,
// matching fast-xml-parser's tolerant behavior. Expansion is bounded:
// DOCTYPE entity values are inserted literally, never re-expanded.
func decodeEntities(s string, doctype map[string]string, html bool) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 || semi > 34 {
			b.WriteByte(s[i])
			i++
			continue
		}
		ent := s[i+1 : i+semi]
		if rep, ok := lookupEntity(ent, doctype, html); ok {
			b.WriteString(rep)
			i += semi + 1
			continue
		}
		b.WriteString(s[i : i+semi+1])
		i += semi + 1
	}
	return b.String()
}

func lookupEntity(ent string, doctype map[string]string, html bool) (string, bool) {
	// fast-xml-parser v5 decodes numeric character references only when
	// htmlEntities is enabled (verified differentially).
	if html && (strings.HasPrefix(ent, "#x") || strings.HasPrefix(ent, "#X")) {
		if n, err := strconv.ParseInt(ent[2:], 16, 32); err == nil {
			return string(rune(n)), true
		}
		return "", false
	}
	if html && strings.HasPrefix(ent, "#") {
		if n, err := strconv.ParseInt(ent[1:], 10, 32); err == nil {
			return string(rune(n)), true
		}
		return "", false
	}
	if v, ok := doctype[ent]; ok {
		return v, true
	}
	if v, ok := xmlEntities[ent]; ok {
		return v, true
	}
	if html {
		if v, ok := htmlEntities[ent]; ok {
			return v, true
		}
	}
	return "", false
}
