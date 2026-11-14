package fxp

import (
	"fmt"
	"strings"
)

// converter turns the ordered node tree into the output structure
// according to Options.
type converter struct {
	opts     *Options
	entities map[string]string
}

func (c *converter) decode(s string) string {
	if !c.opts.ProcessEntities || !strings.Contains(s, "&") {
		return s
	}
	return decodeEntities(s, c.entities, c.opts.HTMLEntities)
}

func (c *converter) tagName(n *node, jPath string) (string, bool) {
	name := n.name
	if c.opts.RemoveNSPrefix && !n.pi {
		if _, local := splitName(name); local != "" {
			name = local
		}
	}
	if c.opts.UpdateTag != nil && !n.pi {
		var ok bool
		name, ok = c.opts.UpdateTag(name, jPath, rawAttrs(n))
		if !ok {
			return "", false
		}
	}
	if c.opts.TransformTagName != nil && !n.pi {
		name = c.opts.TransformTagName(name)
	}
	return name, true
}

func rawAttrs(n *node) string {
	var b strings.Builder
	for i, a := range n.attrs {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(a.name)
		if a.has {
			fmt.Fprintf(&b, "=%q", a.val)
		}
	}
	return b.String()
}

func (c *converter) attrName(a attr) string {
	name := a.name
	if c.opts.RemoveNSPrefix {
		if _, local := splitName(name); local != "" {
			name = local
		}
	}
	if c.opts.TransformAttributeName != nil {
		name = c.opts.TransformAttributeName(name)
	}
	return name
}

// root converts the synthetic root's children into the top-level output:
// map[string]any normally, []any when PreserveOrder is set.
func (c *converter) root(root *node) any {
	if c.opts.PreserveOrder {
		out := []any{}
		for _, ch := range root.children {
			name, keep := c.tagName(ch, ch.name)
			if !keep {
				continue
			}
			wrapper := map[string]any{name: c.orderedChildren(ch, name)}
			if a := c.attrMap(ch); len(a) > 0 {
				wrapper[":@"] = a
			}
			out = append(out, wrapper)
		}
		return out
	}
	out := map[string]any{}
	for _, ch := range root.children {
		name, keep := c.tagName(ch, ch.name)
		if !keep {
			continue
		}
		val := c.nodeValue(ch, name)
		addProp(out, name, val, c.opts, name)
	}
	return out
}

// nodeValue computes the value of one element node (non-preserveOrder).
func (c *converter) nodeValue(n *node, jPath string) any {
	attrs := c.attrMap(n)

	text, extra := c.textAndProps(n)
	if c.opts.TagValueProcessor != nil && text != "" {
		text = c.opts.TagValueProcessor(n.name, text)
	}

	hasContent := len(n.children) > 0 || len(attrs) > 0 || len(extra) > 0

	if !hasContent {
		if c.opts.AlwaysCreateTextNode {
			return map[string]any{c.opts.TextNodeName: parseValue(text, c.opts.ParseTagValue, c.opts.ParseTrueNumberOnly)}
		}
		return parseValue(text, c.opts.ParseTagValue, c.opts.ParseTrueNumberOnly)
	}

	m := map[string]any{}
	for k, v := range attrs {
		m[k] = v
	}
	for _, ch := range n.children {
		name, keep := c.tagName(ch, jPath+"."+ch.name)
		if !keep {
			continue
		}
		childPath := jPath + "." + name
		val := c.nodeValue(ch, childPath)
		addProp(m, name, val, c.opts, childPath)
	}
	if text != "" {
		m[c.opts.TextNodeName] = parseValue(text, c.opts.ParseTagValue, c.opts.ParseTrueNumberOnly)
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

// textAndProps returns the merged text value and any cdata/comment
// properties when those are grouped under configured keys. Matching the
// JS engine: each text part is trimmed individually, then entity-decoded
// (so a decoded entity is never re-trimmed), then concatenated with no
// separator. CDATA parts are never trimmed or decoded.
func (c *converter) textAndProps(n *node) (string, map[string]any) {
	var b strings.Builder
	extra := map[string]any{}
	for _, p := range n.parts {
		switch p.kind {
		case kindText:
			v := p.val
			if c.opts.TrimValues {
				v = strings.TrimSpace(v)
			}
			b.WriteString(c.decode(v))
		case kindCdata:
			if c.opts.CdataPropName != "" {
				appendExtra(extra, c.opts.CdataPropName, p.val)
			} else {
				b.WriteString(p.val)
			}
		case kindComment:
			if c.opts.CommentPropName != "" {
				appendExtra(extra, c.opts.CommentPropName, p.val)
			}
		case kindRaw:
			b.WriteString(p.val)
		}
	}
	return b.String(), extra
}

func appendExtra(m map[string]any, key, val string) {
	if ex, ok := m[key]; ok {
		if arr, isArr := ex.([]string); isArr {
			m[key] = append(arr, val)
		} else {
			m[key] = []string{ex.(string), val}
		}
		return
	}
	m[key] = val
}

func (c *converter) attrMap(n *node) map[string]any {
	if c.opts.IgnoreAttributes {
		return nil
	}
	out := map[string]any{}
	group := c.opts.AttributesGroupName != ""
	var grouped map[string]any
	if group {
		grouped = map[string]any{}
	}
	for _, a := range n.attrs {
		if !a.has && !c.opts.AllowBooleanAttributes {
			continue
		}
		if c.opts.RemoveNSPrefix && (a.name == "xmlns" || strings.HasPrefix(a.name, "xmlns:")) {
			continue // namespace declarations are dropped with the prefix
		}
		name := c.attrName(a)
		var val any
		if !a.has {
			val = true
		} else {
			s := a.val
			if c.opts.TrimValues {
				s = strings.TrimSpace(s)
			}
			s = c.decode(s)
			if c.opts.AttributeValueProcessor != nil {
				s = c.opts.AttributeValueProcessor(a.name, s)
			}
			val = parseValue(s, c.opts.ParseAttributeValue, c.opts.ParseTrueNumberOnly)
		}
		if group {
			grouped[c.opts.AttributeNamePrefix+name] = val
		} else {
			out[c.opts.AttributeNamePrefix+name] = val
		}
	}
	if group {
		if len(grouped) > 0 {
			out[c.opts.AttributesGroupName] = grouped
		}
	}
	return out
}

// addProp inserts val under name, collapsing repeats into arrays.
func addProp(m map[string]any, name string, val any, opts *Options, jPath string) {
	forceArr := opts.IsArray != nil && opts.IsArray(name, jPath, isLeafValue(val), false)
	if existing, ok := m[name]; ok {
		if arr, isArr := existing.([]any); isArr {
			m[name] = append(arr, val)
		} else {
			m[name] = []any{existing, val}
		}
		return
	}
	if forceArr {
		m[name] = []any{val}
		return
	}
	m[name] = val
}

func isLeafValue(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return false
	}
	return true
}

// orderedChildren builds the preserveOrder structure: an array of
// single-key maps; a node's attributes sit in a ":@" key of its wrapper.
func (c *converter) orderedChildren(n *node, jPath string) []any {
	out := []any{}

	// interleave parts and children in document order
	pi, ci := 0, 0
	order := n.order
	if order == nil {
		// fallback: parts then children (tree built without order tracking)
	}
	_ = pi
	_ = ci
	_ = order

	emitPart := func(p textPart) {
		if p.kind == kindRaw {
			out = append(out, map[string]any{c.opts.TextNodeName: p.val})
			return
		}
		switch p.kind {
		case kindText:
			v := p.val
			if c.opts.TrimValues {
				v = strings.TrimSpace(v)
			}
			v = c.decode(v)
			if v == "" {
				return
			}
			out = append(out, map[string]any{c.opts.TextNodeName: parseValue(v, c.opts.ParseTagValue, c.opts.ParseTrueNumberOnly)})
		case kindCdata:
			if c.opts.CdataPropName != "" {
				out = append(out, map[string]any{c.opts.CdataPropName: []any{map[string]any{c.opts.TextNodeName: p.val}}})
			} else {
				out = append(out, map[string]any{c.opts.TextNodeName: p.val})
			}
		case kindComment:
			if c.opts.CommentPropName != "" {
				out = append(out, map[string]any{c.opts.CommentPropName: []any{map[string]any{c.opts.TextNodeName: p.val}}})
			}
		}
	}
	emitChild := func(ch *node) {
		name, keep := c.tagName(ch, jPath+"."+ch.name)
		if !keep {
			return
		}
		wrapper := map[string]any{name: c.orderedChildren(ch, jPath+"."+name)}
		if a := c.attrMap(ch); len(a) > 0 {
			wrapper[":@"] = a
		}
		out = append(out, wrapper)
	}

	// declarations and processing instructions with no content emit an
	// empty text node (JS engine behavior)
	if (n.decl || n.pi) && len(n.order) == 0 {
		out = append(out, map[string]any{c.opts.TextNodeName: ""})
	}
	// true document order: walk recorded order
	for _, item := range n.order {
		if item.isChild {
			emitChild(n.children[item.idx])
		} else {
			emitPart(n.parts[item.idx])
		}
	}
	return out
}
