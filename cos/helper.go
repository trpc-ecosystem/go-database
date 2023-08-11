package cos

import (
	"bytes"
	"fmt"
)

// EncodeURIComponent like same function in javascript
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent
// http://www.ecma-international.org/ecma-262/6.0/#sec-uri-syntax-and-semantics
// From TencentCloud COS SDK
func EncodeURIComponent(s string, lower bool, excluded ...[]byte) string {
	var b bytes.Buffer
	written := 0

	for i, n := 0, len(s); i < n; i++ {
		c := s[i]

		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if isNumOrLetter(c) {
				continue
			}

			if len(excluded) > 0 && isExcluded(c, excluded...) {
				continue
			}
		}

		b.WriteString(s[written:i])
		if lower {
			fmt.Fprintf(&b, "%%%02x", c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
		written = i + 1
	}

	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}

func isNumOrLetter(c uint8) (ret bool) {
	if 'a' <= c && c <= 'z' {
		return true
	}
	if 'A' <= c && c <= 'Z' {
		return true
	}
	if '0' <= c && c <= '9' {
		return true
	}

	return false
}

func isExcluded(c uint8, excluded ...[]byte) (ret bool) {
	if len(excluded) == 0 {
		return false
	}
	for _, ch := range excluded[0] {
		if ch == c {
			return true
		}
	}

	return
}
