package fxp

import (
	"fmt"
	"strings"
)

// Parser parses XML into JSON-compatible map structures.
// A Parser is not safe for concurrent use.
type Parser struct {
	opts Options

	// per-Parse scanner state
	data     string
	pos      int
	entities map[string]string
}

// New returns a Parser. Pass DefaultOptions() (mutated as needed),
// or Options{} for the minimal zero-value configuration.
func New(opts Options) *Parser {
	if opts.AttributeNamePrefix == "" {
		opts.AttributeNamePrefix = "@_"
	}
	if opts.TextNodeName == "" {
		opts.TextNodeName = "#text"
	}
	return &Parser{opts: opts}
}

// Parse parses xml. It returns map[string]any normally, or []any when
// Options.PreserveOrder is set. Values are string, float64, bool,
// map[string]any, or []any.
func (p *Parser) Parse(xml string) (any, error) {
	root, err := p.scan(xml)
	if err != nil {
		return nil, err
	}
	c := &converter{opts: &p.opts, entities: p.entities}
	return c.root(root), nil
}

// scanner state held on Parser only for the duration of one Parse call;
// Parser is not safe for concurrent use.
func (p *Parser) scan(data string) (*node, error) {
	p.entities = map[string]string{}
	p.pos = 0
	p.data = data

	// Pre-scan DOCTYPE internal subset for entity definitions.
	if p.opts.ProcessEntities {
		p.scanDoctype(data)
	}

	root := &node{name: ""}
	stack := []*node{root}
	path := []string{}

	unpaired := map[string]bool{}
	for _, t := range p.opts.UnpairedTags {
		unpaired[strings.ToLower(t)] = true
	}
	stops := map[string]bool{}
	for _, s := range p.opts.StopNodes {
		stops[s] = true
	}

	for p.pos < len(p.data) {
		if p.data[p.pos] != '<' {
			end := strings.IndexByte(p.data[p.pos:], '<')
			if end < 0 {
				end = len(p.data) - p.pos
			}
			text := p.data[p.pos : p.pos+end]
			p.pos += end
			cur := stack[len(stack)-1]
			cur.addText(kindText, text) // raw; entities decode at conversion, after trimming
			continue
		}
		rest := p.data[p.pos:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest[4:], "-->")
			if end < 0 {
				return nil, p.errHere("unclosed comment")
			}
			comment := rest[4 : 4+end]
			p.pos += 4 + end + 3
			cur := stack[len(stack)-1]
			cur.addText(kindComment, comment)
		case strings.HasPrefix(rest, "<![CDATA["):
			end := strings.Index(rest[9:], "]]>")
			if end < 0 {
				return nil, p.errHere("unclosed CDATA")
			}
			cd := rest[9 : 9+end]
			p.pos += 9 + end + 3
			cur := stack[len(stack)-1]
			cur.addText(kindCdata, cd)
		case strings.HasPrefix(rest, "<!"):
			// DOCTYPE or other markup declaration: skip to '>' (naive; subset pre-scanned)
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return nil, p.errHere("unclosed markup declaration")
			}
			p.pos += end + 1
		case strings.HasPrefix(rest, "<?"):
			end := strings.Index(rest[2:], "?>")
			if end < 0 {
				return nil, p.errHere("unclosed processing instruction")
			}
			end += 2
			inner := rest[2:end]
			p.pos += end + 2
			cur := stack[len(stack)-1]
			n := parsePiOrDecl(inner)
			if n.decl {
				if p.opts.IgnoreDeclaration {
					continue
				}
			} else if p.opts.IgnorePiTags {
				continue
			}
			cur.addChild(n)
		case strings.HasPrefix(rest, "</"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return nil, p.errHere("unclosed closing tag")
			}
			name := strings.TrimSpace(rest[2:end])
			p.pos += end + 1
			// pop to matching open tag (tolerant of mismatches)
			found := -1
			for i := len(stack) - 1; i >= 1; i-- {
				if stack[i].name == name {
					found = i
					break
				}
			}
			if found < 0 {
				continue // unmatched close: tolerate
			}
			stack = stack[:found]
			path = path[:len(stack)-1]
		default:
			// opening or self-closing tag
			n, raw, err := p.scanTag(rest, stack, path, stops)
			if err != nil {
				return nil, err
			}
			if raw {
				continue // stop node consumed raw content
			}
			if n == nil {
				continue
			}
			name := n.name
			if unpaired[strings.ToLower(name)] {
				// unpaired: attach but never push
				continue
			}
			if !n.selfClos {
				if p.opts.MaxNestedTags > 0 && len(stack) > p.opts.MaxNestedTags {
					return nil, fmt.Errorf("fxp: maximum nested tags (%d) exceeded at <%s>", p.opts.MaxNestedTags, name)
				}
				stack = append(stack, n)
				path = append(path, name)
			}
		}
	}
	return root, nil
}

// scanTag parses an opening tag starting at "<". If the full path is a
// stop node, captures raw content until the matching close and returns raw=true.
func (p *Parser) scanTag(rest string, stack []*node, path []string, stops map[string]bool) (*node, bool, error) {
	i := 1
	// tag name
	start := i
	for i < len(rest) && !isSpace(rest[i]) && rest[i] != '>' && rest[i] != '/' {
		i++
	}
	if i == start {
		return nil, false, p.errHere("empty tag name")
	}
	name := rest[start:i]
	fullPath := strings.Join(append(append([]string{}, path...), name), ".")

	if stops[fullPath] {
		// Capture raw content until the matching close tag, matching the
		// JS engine: nested "<tag>" is tolerated, but "<" followed by an
		// invalid tag-name character errors with "Unexpected end".
		raw := rest[i:]
		gt := strings.IndexByte(raw, '>')
		if gt < 0 {
			return nil, false, p.errHere("unclosed stop node " + name)
		}
		content := raw[gt+1:]
		scan := 0
		for scan < len(content) {
			lt := strings.IndexByte(content[scan:], '<')
			if lt < 0 {
				return nil, false, p.errHere("Unexpected end of " + name)
			}
			at := scan + lt
			if at+1 >= len(content) {
				return nil, false, p.errHere("Unexpected end of " + name)
			}
			nx := content[at+1]
			if nx == '/' {
				if strings.HasPrefix(content[at:], "</"+name+">") {
					cur := stack[len(stack)-1]
					n := cur.child(name)
					n.addText(kindRaw, content[:at])
					p.pos += i + gt + 1 + at + len(name) + 3
					return nil, true, nil
				}
				scan = at + 2
				continue
			}
			if isTagNameStart(nx) {
				// nested tag open: skip to its '>'
				end := strings.IndexByte(content[at:], '>')
				if end < 0 {
					return nil, false, p.errHere("Unexpected end of " + name)
				}
				scan = at + end + 1
				continue
			}
			return nil, false, p.errHere("Unexpected end of " + name)
		}
		return nil, false, p.errHere("Unexpected end of " + name)
	}

	n := &node{name: name}
	cur := stack[len(stack)-1]
	cur.addChild(n)

	// attributes
	for i < len(rest) {
		for i < len(rest) && isSpace(rest[i]) {
			i++
		}
		if i >= len(rest) {
			return nil, false, p.errHere("unclosed tag " + name)
		}
		if rest[i] == '>' {
			i++
			break
		}
		if rest[i] == '/' {
			if i+1 < len(rest) && rest[i+1] == '>' {
				n.selfClos = true
				i += 2
				break
			}
			i++
			continue
		}
		// attribute name
		aStart := i
		for i < len(rest) && !isSpace(rest[i]) && rest[i] != '=' && rest[i] != '>' && rest[i] != '/' {
			i++
		}
		aName := rest[aStart:i]
		for i < len(rest) && isSpace(rest[i]) {
			i++
		}
		if i < len(rest) && rest[i] == '=' {
			i++
			for i < len(rest) && isSpace(rest[i]) {
				i++
			}
			if i >= len(rest) || (rest[i] != '"' && rest[i] != '\'') {
				return nil, false, p.errHere("attribute " + aName + " missing quoted value")
			}
			q := rest[i]
			i++
			vStart := i
			for i < len(rest) && rest[i] != q {
				i++
			}
			if i >= len(rest) {
				return nil, false, p.errHere("unclosed attribute value")
			}
			val := rest[vStart:i]
			i++
			n.attrs = append(n.attrs, attr{name: aName, val: val, has: true})
		} else {
			n.attrs = append(n.attrs, attr{name: aName})
		}
	}
	p.pos += i
	return n, false, nil
}

func parsePiOrDecl(inner string) *node {
	inner = strings.TrimSpace(inner)
	i := 0
	for i < len(inner) && !isSpace(inner[i]) && inner[i] != '?' {
		i++
	}
	target := inner[:i]
	n := &node{name: "?" + target, pi: true, decl: target == "xml"}
	// parse pseudo-attributes
	rest := inner[i:]
	for len(rest) > 0 {
		rest = strings.TrimLeft(rest, " \t\r\n")
		if rest == "" || rest == "?" {
			break
		}
		j := 0
		for j < len(rest) && rest[j] != '=' && !isSpace(rest[j]) {
			j++
		}
		aName := rest[:j]
		rest = rest[j:]
		rest = strings.TrimLeft(rest, " \t\r\n")
		if rest == "" || rest[0] != '=' {
			n.attrs = append(n.attrs, attr{name: aName})
			continue
		}
		rest = strings.TrimLeft(rest[1:], " \t\r\n")
		if rest == "" {
			break
		}
		q := rest[0]
		if q != '"' && q != '\'' {
			break
		}
		k := strings.IndexByte(rest[1:], q)
		if k < 0 {
			break
		}
		n.attrs = append(n.attrs, attr{name: aName, val: rest[1 : 1+k], has: true})
		rest = rest[1+k+1:]
	}
	return n
}

// scanDoctype extracts ENTITY definitions from a DOCTYPE internal subset.
func (p *Parser) scanDoctype(data string) {
	idx := strings.Index(data, "<!DOCTYPE")
	if idx < 0 {
		return
	}
	rest := data[idx:]
	open := strings.IndexByte(rest, '[')
	close := strings.Index(rest, "]>")
	if open < 0 || close < 0 || close < open {
		return
	}
	subset := rest[open+1 : close]
	for {
		e := strings.Index(subset, "<!ENTITY")
		if e < 0 {
			return
		}
		subset = subset[e+8:]
		gt := strings.IndexByte(subset, '>')
		if gt < 0 {
			return
		}
		def := strings.TrimSpace(subset[:gt])
		subset = subset[gt+1:]
		// forms: name "value" | name 'value' ; skip parameter entities (%)
		if strings.HasPrefix(def, "%") {
			continue
		}
		var name, val string
		if i := strings.IndexAny(def, "\"'"); i > 0 {
			name = strings.TrimSpace(def[:i])
			q := def[i]
			body := def[i+1:]
			if j := strings.IndexByte(body, q); j >= 0 {
				val = body[:j]
			}
		}
		if name != "" {
			p.entities[name] = val
		}
	}
}

func (p *Parser) decodeEntities(s string) string {
	if !p.opts.ProcessEntities || !strings.Contains(s, "&") {
		return s
	}
	return decodeEntities(s, p.entities, p.opts.HTMLEntities)
}

func (p *Parser) errHere(msg string) error {
	line := 1 + strings.Count(p.data[:p.pos], "\n")
	return fmt.Errorf("fxp: %s (line %d)", msg, line)
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

func isTagNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
