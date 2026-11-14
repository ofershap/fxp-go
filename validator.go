package fxp

import (
	"fmt"
	"strings"
)

// Validate checks XML well-formedness strictly (unlike Parse, which is
// tolerant). It returns nil for valid XML, else an error with the line.
// Matches the scope of fast-xml-parser's XMLValidator: tag balance,
// attribute quoting, and tag-name syntax; it is not a schema validator.
func Validate(xml string) error {
	pos := 0
	type openTag struct {
		name string
		line int
	}
	var stack []openTag
	lineAt := func(p int) int { return 1 + strings.Count(xml[:p], "\n") }

	for pos < len(xml) {
		if xml[pos] != '<' {
			nxt := strings.IndexByte(xml[pos:], '<')
			if nxt < 0 {
				break
			}
			pos += nxt
			continue
		}
		rest := xml[pos:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest[4:], "-->")
			if end < 0 {
				return fmt.Errorf("fxp: unclosed comment (line %d)", lineAt(pos))
			}
			pos += 4 + end + 3
		case strings.HasPrefix(rest, "<![CDATA["):
			end := strings.Index(rest[9:], "]]>")
			if end < 0 {
				return fmt.Errorf("fxp: unclosed CDATA (line %d)", lineAt(pos))
			}
			pos += 9 + end + 3
		case strings.HasPrefix(rest, "<!"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return fmt.Errorf("fxp: unclosed markup declaration (line %d)", lineAt(pos))
			}
			pos += end + 1
		case strings.HasPrefix(rest, "<?"):
			end := strings.Index(rest, "?>")
			if end < 0 {
				return fmt.Errorf("fxp: unclosed processing instruction (line %d)", lineAt(pos))
			}
			pos += end + 2
		case strings.HasPrefix(rest, "</"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return fmt.Errorf("fxp: unclosed closing tag (line %d)", lineAt(pos))
			}
			name := strings.TrimSpace(rest[2:end])
			if name == "" || !validTagName(name) {
				return fmt.Errorf("fxp: invalid closing tag (line %d)", lineAt(pos))
			}
			if len(stack) == 0 {
				return fmt.Errorf("fxp: closing tag </%s> without opening tag (line %d)", name, lineAt(pos))
			}
			top := stack[len(stack)-1]
			if top.name != name {
				return fmt.Errorf("fxp: expected closing tag </%s> (opened line %d), got </%s> (line %d)", top.name, top.line, name, lineAt(pos))
			}
			stack = stack[:len(stack)-1]
			pos += end + 1
		default:
			// opening tag
			i := 1
			start := i
			for i < len(rest) && !isSpace(rest[i]) && rest[i] != '>' && rest[i] != '/' {
				i++
			}
			name := rest[start:i]
			if name == "" || !validTagName(name) {
				return fmt.Errorf("fxp: invalid tag name (line %d)", lineAt(pos))
			}
			selfClose := false
			for i < len(rest) {
				for i < len(rest) && isSpace(rest[i]) {
					i++
				}
				if i >= len(rest) {
					return fmt.Errorf("fxp: unclosed tag <%s> (line %d)", name, lineAt(pos))
				}
				if rest[i] == '>' {
					i++
					break
				}
				if rest[i] == '/' {
					if i+1 < len(rest) && rest[i+1] == '>' {
						selfClose = true
						i += 2
						break
					}
					return fmt.Errorf("fxp: invalid '/' in tag <%s> (line %d)", name, lineAt(pos))
				}
				aStart := i
				for i < len(rest) && !isSpace(rest[i]) && rest[i] != '=' && rest[i] != '>' && rest[i] != '/' {
					i++
				}
				if i == aStart {
					return fmt.Errorf("fxp: invalid attribute in <%s> (line %d)", name, lineAt(pos))
				}
				for i < len(rest) && isSpace(rest[i]) {
					i++
				}
				if i < len(rest) && rest[i] == '=' {
					i++
					for i < len(rest) && isSpace(rest[i]) {
						i++
					}
					if i >= len(rest) || (rest[i] != '"' && rest[i] != '\'') {
						return fmt.Errorf("fxp: attribute value must be quoted in <%s> (line %d)", name, lineAt(pos))
					}
					q := rest[i]
					i++
					end := strings.IndexByte(rest[i:], q)
					if end < 0 {
						return fmt.Errorf("fxp: unclosed attribute value in <%s> (line %d)", name, lineAt(pos))
					}
					i += end + 1
				}
				// boolean attribute: allowed by the validator (matches FXP)
			}
			if !selfClose {
				stack = append(stack, openTag{name, lineAt(pos)})
			}
			pos += i
		}
	}
	if len(stack) > 0 {
		top := stack[len(stack)-1]
		return fmt.Errorf("fxp: unclosed tag <%s> (opened line %d)", top.name, top.line)
	}
	return nil
}

func validTagName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		ok := c == '_' || c == '-' || c == '.' || c == ':' ||
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(i > 0 && c >= '0' && c <= '9')
		if !ok {
			return false
		}
	}
	return true
}
