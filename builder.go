package fxp

import (
	"fmt"
	"sort"
	"strings"
)

// BuildOptions mirrors the fast-xml-parser v5.x XMLBuilder option surface.
type BuildOptions struct {
	// AttributeNamePrefix identifies attribute keys in input. Default "@_".
	AttributeNamePrefix string
	// AttributesGroupName groups attributes under one key.
	AttributesGroupName string
	// TextNodeName names the text property. Default "#text".
	TextNodeName string
	// CdataPropName names the CDATA property.
	CdataPropName string
	// CommentPropName names the comment property.
	CommentPropName string
	// IgnoreAttributes skips attribute keys. Default false.
	IgnoreAttributes bool
	// ProcessEntities escapes < > & in text and < > & " in attributes. Default true.
	ProcessEntities bool
	// Format pretty-prints with indentation. Default false.
	Format bool
	// IndentBy is the indentation unit when Format is set. Default "  ".
	IndentBy string
	// SuppressEmptyNode renders empty elements as <tag></tag> instead of <tag/>. Default false.
	SuppressEmptyNode bool
	// SuppressBooleanAttributes renders true-valued attributes without ="...". Default false.
	SuppressBooleanAttributes bool
	// PreserveOrder consumes the ordered array structure. Default false.
	PreserveOrder bool
	// UnpairedTags renders these tags without a closing tag.
	UnpairedTags []string
	// StopNodes renders these tag paths' content raw (unescaped).
	StopNodes []string
	// OneListGroup, ArrayNodeName: reserved, not yet implemented.
}

// DefaultBuildOptions returns the fast-xml-parser v5.x builder defaults.
func DefaultBuildOptions() BuildOptions {
	return BuildOptions{
		AttributeNamePrefix: "@_",
		TextNodeName:        "#text",
		ProcessEntities:     true,
	}
}

// Builder builds XML from the map structure produced by Parser.
type Builder struct {
	opts BuildOptions
}

// NewBuilder returns a Builder with the given options.
func NewBuilder(opts BuildOptions) *Builder {
	if opts.AttributeNamePrefix == "" {
		opts.AttributeNamePrefix = "@_"
	}
	if opts.TextNodeName == "" {
		opts.TextNodeName = "#text"
	}
	return &Builder{opts: opts}
}

// Build converts v into an XML document string.
func (b *Builder) Build(v any) (string, error) {
	var out strings.Builder
	if b.opts.PreserveOrder {
		arr, ok := v.([]any)
		if !ok {
			return "", fmt.Errorf("fxp: preserveOrder build expects []any, got %T", v)
		}
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if err := b.buildOrdered(&out, m, "", 0); err != nil {
				return "", err
			}
		}
		return out.String(), nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("fxp: build expects map[string]any, got %T", v)
	}
	for _, k := range sortedKeys(m) {
		if err := b.buildNode(&out, k, m[k], k, 0); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysStr(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (b *Builder) buildNode(out *strings.Builder, name string, val any, path string, depth int) error {
	switch x := val.(type) {
	case []any:
		for _, item := range x {
			if err := b.buildNode(out, name, item, path, depth); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		attrs, children, text := b.splitMap(x)
		if strings.HasPrefix(name, "?") {
			b.piTag(out, name, attrs, depth)
			return nil
		}
		b.openTag(out, name, attrs, len(children) == 0 && text == "", depth)
		if text != "" {
			b.indent(out, depth+1, true)
			out.WriteString(b.escapeText(text))
			b.indent(out, depth, len(children) > 0)
		}
		for _, k := range children {
			if b.opts.CdataPropName != "" && k == b.opts.CdataPropName {
				for _, v := range flatValues(x[k]) {
					out.WriteString("<![CDATA[")
					out.WriteString(scalarString(v))
					out.WriteString("]]>")
				}
				continue
			}
			if b.opts.CommentPropName != "" && k == b.opts.CommentPropName {
				for _, v := range flatValues(x[k]) {
					out.WriteString("<!--")
					out.WriteString(scalarString(v))
					out.WriteString("-->")
				}
				continue
			}
			if err := b.buildNode(out, k, x[k], path+"."+k, depth+1); err != nil {
				return err
			}
		}
		if len(children) == 0 && text == "" {
			if b.opts.SuppressEmptyNode {
				out.WriteString("</" + name + ">")
			}
			return nil
		}
		if b.opts.Format && len(children) > 0 {
			b.indent(out, depth, true)
		}
		out.WriteString("</" + name + ">")
		return nil
	default:
		s := scalarString(x)
		if strings.HasPrefix(name, "?") {
			b.piTag(out, name, nil, depth)
			return nil
		}
		b.openTag(out, name, nil, s == "", depth)
		out.WriteString(b.escapeText(s))
		if s == "" && !b.opts.SuppressEmptyNode {
			return nil
		}
		out.WriteString("</" + name + ">")
		return nil
	}
}

// splitMap separates attribute keys, child element keys and the text value.
func (b *Builder) splitMap(m map[string]any) (attrs map[string]string, children []string, text string) {
	attrs = map[string]string{}
	for _, k := range sortedKeys(m) {
		v := m[k]
		switch {
		case k == b.opts.TextNodeName:
			text = scalarString(v)
		case b.opts.CdataPropName != "" && k == b.opts.CdataPropName:
			children = append(children, k)
		case b.opts.CommentPropName != "" && k == b.opts.CommentPropName:
			children = append(children, k)
		case b.opts.AttributesGroupName != "" && k == b.opts.AttributesGroupName:
			if gm, ok := v.(map[string]any); ok {
				for _, gk := range sortedKeys(gm) {
					name := strings.TrimPrefix(gk, b.opts.AttributeNamePrefix)
					if gv, ok := gm[gk].(bool); ok && gv && b.opts.SuppressBooleanAttributes {
						attrs[name] = "\x00bool"
					} else {
						attrs[name] = scalarString(gm[gk])
					}
				}
			}
		case strings.HasPrefix(k, b.opts.AttributeNamePrefix):
			if b.opts.IgnoreAttributes {
				continue
			}
			name := strings.TrimPrefix(k, b.opts.AttributeNamePrefix)
			if bv, ok := v.(bool); ok && bv && b.opts.SuppressBooleanAttributes {
				attrs[name] = "\x00bool"
			} else {
				attrs[name] = scalarString(v)
			}
		default:
			children = append(children, k)
		}
	}
	return attrs, children, text
}

// piTag renders processing instructions and declarations: <?name attrs?>.
func (b *Builder) piTag(out *strings.Builder, name string, attrs map[string]string, depth int) {
	b.indent(out, depth, false)
	out.WriteByte('<')
	out.WriteString(name)
	for _, k := range sortedKeysStr(attrs) {
		out.WriteByte(' ')
		out.WriteString(k)
		if attrs[k] == "\x00bool" {
			continue
		}
		out.WriteString(`="`)
		out.WriteString(b.escapeAttr(attrs[k]))
		out.WriteByte('"')
	}
	out.WriteString("?>")
}

func (b *Builder) openTag(out *strings.Builder, name string, attrs map[string]string, empty bool, depth int) {
	b.indent(out, depth, false)
	out.WriteByte('<')
	out.WriteString(name)
	for _, k := range sortedKeysStr(attrs) {
		out.WriteByte(' ')
		out.WriteString(k)
		if attrs[k] == "\x00bool" {
			continue
		}
		out.WriteString(`="`)
		out.WriteString(b.escapeAttr(attrs[k]))
		out.WriteByte('"')
	}
	if empty && !b.opts.SuppressEmptyNode {
		out.WriteString("/>")
	} else {
		out.WriteByte('>')
	}
}

func (b *Builder) indent(out *strings.Builder, depth int, newline bool) {
	if !b.opts.Format {
		return
	}
	if newline {
		out.WriteByte('\n')
	}
	unit := b.opts.IndentBy
	if unit == "" {
		unit = "  "
	}
	for i := 0; i < depth; i++ {
		out.WriteString(unit)
	}
}

func (b *Builder) buildOrdered(out *strings.Builder, m map[string]any, path string, depth int) error {
	attrs, _ := m[":@"].(map[string]any)
	attrStr := map[string]string{}
	for _, k := range sortedKeys(attrs) {
		name := strings.TrimPrefix(k, b.opts.AttributeNamePrefix)
		if bv, ok := attrs[k].(bool); ok && bv && b.opts.SuppressBooleanAttributes {
			attrStr[name] = "\x00bool"
		} else {
			attrStr[name] = scalarString(attrs[k])
		}
	}
	for _, k := range sortedKeys(m) {
		if k == ":@" {
			continue
		}
		val := m[k]
		// text / cdata / comment pseudo nodes
		if k == b.opts.TextNodeName {
			out.WriteString(b.escapeText(scalarString(val)))
			continue
		}
		if b.opts.CdataPropName != "" && k == b.opts.CdataPropName {
			out.WriteString("<![CDATA[")
			out.WriteString(cdataText(val, b.opts.TextNodeName))
			out.WriteString("]]>")
			continue
		}
		if b.opts.CommentPropName != "" && k == b.opts.CommentPropName {
			out.WriteString("<!--")
			out.WriteString(cdataText(val, b.opts.TextNodeName))
			out.WriteString("-->")
			continue
		}
		children, _ := val.([]any)
		b.indent(out, depth, false)
		b.openTag(out, k, attrStr, false, 0)
		for _, item := range children {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if err := b.buildOrdered(out, im, path+"."+k, depth+1); err != nil {
				return err
			}
		}
		if isUnpaired(b.opts.UnpairedTags, k) {
			continue
		}
		out.WriteString("</" + k + ">")
	}
	return nil
}

func flatValues(v any) []any {
	switch arr := v.(type) {
	case []any:
		return arr
	case []string:
		out := make([]any, len(arr))
		for i, s := range arr {
			out[i] = s
		}
		return out
	}
	return []any{v}
}

func cdataText(v any, textNodeName string) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		for _, item := range x {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m[textNodeName].(string); ok {
					return t
				}
			}
		}
	}
	return ""
}

func isUnpaired(list []string, name string) bool {
	for _, t := range list {
		if strings.EqualFold(t, name) {
			return true
		}
	}
	return false
}

func scalarString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return formatNumber(x)
	case []any:
		if len(x) == 0 {
			return ""
		}
		return scalarString(x[0])
	}
	return fmt.Sprint(v)
}

// formatNumber matches JS String(number): integers without decimal point.
func formatNumber(f float64) string {
	if f == float64(int64(f)) && f < 1e21 && f > -1e21 {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}

func (b *Builder) escapeText(s string) string {
	if !b.opts.ProcessEntities {
		return s
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func (b *Builder) escapeAttr(s string) string {
	if !b.opts.ProcessEntities {
		return s
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
