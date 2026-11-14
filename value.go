package fxp

import "strconv"

// parseValue converts a raw string per FXP's parseTagValue rules:
// "true"/"false" to bool, strict numeric strings to float64, else the
// string unchanged. Hex (0x1A) and leading zeros (0123 -> 123) match the
// JS engine's default numberParseOptions; Infinity/NaN stay strings.
func parseValue(s string, parse bool, trueNumberOnly bool) any {
	if !parse {
		return s
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if s == "" {
		return s
	}
	if n, ok := parseNumber(s); ok {
		return n
	}
	return s
}

func parseNumber(s string) (float64, bool) {
	if !isStrictNumber(s) {
		return 0, false
	}
	neg := false
	body := s
	if body[0] == '+' || body[0] == '-' {
		neg = body[0] == '-'
		body = body[1:]
	}
	if len(body) > 2 && (body[:2] == "0x" || body[:2] == "0X") {
		v, err := strconv.ParseInt(body[2:], 16, 64)
		if err != nil {
			return 0, false
		}
		f := float64(v)
		if neg {
			f = -f
		}
		return f, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// isStrictNumber matches ^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$ or
// ^[+-]?0[xX][0-9a-fA-F]+$ - the shapes JS Number() accepts that FXP parses.
func isStrictNumber(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	if i < len(s)-1 && s[i] == '0' && (s[i+1] == 'x' || s[i+1] == 'X') {
		if i+2 == len(s) {
			return false
		}
		for _, c := range s[i+2:] {
			if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
				return false
			}
		}
		return true
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		ed := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			ed++
		}
		if ed == 0 {
			return false
		}
	}
	return i == len(s)
}
