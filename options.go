package fxp

// Options mirrors the fast-xml-parser v5.x XMLParser option surface.
// Use DefaultOptions() and mutate the fields you need: several FXP
// defaults are true, which a zero-value struct cannot express.
type Options struct {
	// AttributeNamePrefix identifies attributes in output. Default "@_".
	AttributeNamePrefix string
	// AttributesGroupName groups all attributes under one key instead of
	// prefixing them at the node level.
	AttributesGroupName string
	// TextNodeName names the property holding mixed text content. Default "#text".
	TextNodeName string
	// IgnoreAttributes drops all attributes. Default true.
	IgnoreAttributes bool
	// IgnorePiTags drops processing instructions. Default false.
	IgnorePiTags bool
	// IgnoreDeclaration drops the <?xml ...?> declaration. Default false.
	IgnoreDeclaration bool
	// ParseTagValue converts tag text like "123" or "true" to number/bool. Default true.
	ParseTagValue bool
	// ParseAttributeValue converts attribute values to number/bool. Default false.
	ParseAttributeValue bool
	// ParseTrueNumberOnly only converts strings that fully match a number. Default false.
	ParseTrueNumberOnly bool
	// TrimValues trims surrounding whitespace of values. Default true.
	TrimValues bool
	// CdataPropName groups CDATA under this key; empty merges CDATA into text.
	CdataPropName string
	// CommentPropName groups comments under this key; empty drops comments.
	CommentPropName string
	// PreserveOrder outputs children as ordered arrays of single-key maps
	// with attributes under ":@" instead of grouped properties. Default false.
	PreserveOrder bool
	// ProcessEntities decodes XML/HTML entities and DOCTYPE-defined entities. Default true.
	ProcessEntities bool
	// HTMLEntities also decodes HTML named entities. Default false.
	HTMLEntities bool
	// RemoveNSPrefix strips namespace prefixes from tag and attribute names. Default false.
	RemoveNSPrefix bool
	// AllowBooleanAttributes parses valueless attributes as true. Default false.
	AllowBooleanAttributes bool
	// AlwaysCreateTextNode wraps every leaf value under TextNodeName. Default false.
	AlwaysCreateTextNode bool
	// StopNodes lists tag paths (e.g. "root.script") whose raw content is kept unparsed.
	StopNodes []string
	// UnpairedTags lists tag names closed implicitly (e.g. "br"). Default empty.
	UnpairedTags []string
	// MaxNestedTags limits nesting depth; 0 means unlimited.
	MaxNestedTags int
	// IsArray forces array output for matching tag/jpath.
	IsArray func(tagName, jPath string, isLeaf, isAttr bool) bool
	// TagValueProcessor transforms a tag's parsed value.
	TagValueProcessor func(tagName, val string) string
	// AttributeValueProcessor transforms an attribute value.
	AttributeValueProcessor func(attrName, val string) string
	// TransformTagName transforms tag names in output.
	TransformTagName func(tagName string) string
	// TransformAttributeName transforms attribute names in output.
	TransformAttributeName func(attrName string) string
	// UpdateTag renames or drops tags during parsing. Return false to drop.
	UpdateTag func(tagName, jPath, attrs string) (string, bool)
}

// DefaultOptions returns the fast-xml-parser v5.x defaults.
func DefaultOptions() Options {
	return Options{
		AttributeNamePrefix: "@_",
		TextNodeName:        "#text",
		IgnoreAttributes:    true,
		ParseTagValue:       true,
		TrimValues:          true,
		ProcessEntities:     true,
	}
}
