package fxp

import "strings"

// node is the intermediate ordered tree the lexer builds before
// option-driven conversion to the output map.
type node struct {
	name     string
	attrs    []attr      // in document order
	children []*node     // element children in order
	parts    []textPart  // text/cdata/comment in order
	order    []orderItem // interleaved document order of children and parts
	selfClos bool
	pi       bool // processing instruction
	decl     bool // <?xml ...?> declaration
}

type orderItem struct {
	isChild bool
	idx     int
}

type attr struct {
	name string
	val  string
	has  bool // false for boolean (valueless) attributes
}

type textKind int

const (
	kindText textKind = iota
	kindCdata
	kindComment
)

type textPart struct {
	kind textKind
	val  string
}

func (n *node) addText(kind textKind, val string) {
	if val == "" {
		return
	}
	if kind == kindText && len(n.order) > 0 {
		last := n.order[len(n.order)-1]
		if !last.isChild && n.parts[last.idx].kind == kindText {
			n.parts[last.idx].val += val
			return
		}
	}
	n.order = append(n.order, orderItem{false, len(n.parts)})
	n.parts = append(n.parts, textPart{kind, val})
}

func (n *node) child(name string) *node {
	c := &node{name: name}
	n.order = append(n.order, orderItem{true, len(n.children)})
	n.children = append(n.children, c)
	return c
}

func (n *node) addChild(c *node) {
	n.order = append(n.order, orderItem{true, len(n.children)})
	n.children = append(n.children, c)
}

func splitName(name string) (ns, local string) {
	if i := strings.IndexByte(name, ':'); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "", name
}
