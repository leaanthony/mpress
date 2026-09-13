package mpd

const maxJSONDepth = 256

// scanJSONValue validates one JSON value and returns the first byte after it.
// It stops before attribute-separating whitespace but accepts whitespace inside
// arrays and objects. The implementation is allocation-free.
func scanJSONValue(source []byte, start int) (int, bool) {
	cursor := jsonCursor{source: source, offset: start}
	if !cursor.value(0) {
		return cursor.offset, false
	}
	return cursor.offset, true
}

type jsonCursor struct {
	source []byte
	offset int
}

func (c *jsonCursor) value(depth int) bool {
	if depth > maxJSONDepth || c.offset >= len(c.source) {
		return false
	}
	switch c.source[c.offset] {
	case '"':
		return c.string()
	case '[':
		return c.array(depth + 1)
	case '{':
		return c.object(depth + 1)
	case 't':
		return c.literal("true")
	case 'f':
		return c.literal("false")
	case 'n':
		return c.literal("null")
	case '-':
		return c.number()
	default:
		if c.source[c.offset] >= '0' && c.source[c.offset] <= '9' {
			return c.number()
		}
		return false
	}
}

func (c *jsonCursor) string() bool {
	c.offset++
	for c.offset < len(c.source) {
		char := c.source[c.offset]
		if char == '"' {
			c.offset++
			return true
		}
		if char < 0x20 {
			return false
		}
		if char != '\\' {
			c.offset++
			continue
		}
		c.offset++
		if c.offset >= len(c.source) {
			return false
		}
		escape := c.source[c.offset]
		if escape == 'u' {
			if c.offset+4 >= len(c.source) {
				return false
			}
			for index := c.offset + 1; index <= c.offset+4; index++ {
				if !isHex(c.source[index]) {
					return false
				}
			}
			c.offset += 5
			continue
		}
		if escape != '"' && escape != '\\' && escape != '/' && escape != 'b' && escape != 'f' && escape != 'n' && escape != 'r' && escape != 't' {
			return false
		}
		c.offset++
	}
	return false
}

func (c *jsonCursor) array(depth int) bool {
	c.offset++
	c.space()
	if c.take(']') {
		return true
	}
	for {
		if !c.value(depth) {
			return false
		}
		c.space()
		if c.take(']') {
			return true
		}
		if !c.take(',') {
			return false
		}
		c.space()
	}
}

func (c *jsonCursor) object(depth int) bool {
	c.offset++
	c.space()
	if c.take('}') {
		return true
	}
	for {
		if c.offset >= len(c.source) || c.source[c.offset] != '"' || !c.string() {
			return false
		}
		c.space()
		if !c.take(':') {
			return false
		}
		c.space()
		if !c.value(depth) {
			return false
		}
		c.space()
		if c.take('}') {
			return true
		}
		if !c.take(',') {
			return false
		}
		c.space()
	}
}

func (c *jsonCursor) number() bool {
	if c.take('-') && c.offset >= len(c.source) {
		return false
	}
	if c.take('0') {
		if c.offset < len(c.source) && c.source[c.offset] >= '0' && c.source[c.offset] <= '9' {
			return false
		}
	} else {
		start := c.offset
		for c.offset < len(c.source) && c.source[c.offset] >= '0' && c.source[c.offset] <= '9' {
			c.offset++
		}
		if c.offset == start {
			return false
		}
	}
	if c.take('.') {
		start := c.offset
		for c.offset < len(c.source) && c.source[c.offset] >= '0' && c.source[c.offset] <= '9' {
			c.offset++
		}
		if c.offset == start {
			return false
		}
	}
	if c.offset < len(c.source) && (c.source[c.offset] == 'e' || c.source[c.offset] == 'E') {
		c.offset++
		if c.offset < len(c.source) && (c.source[c.offset] == '+' || c.source[c.offset] == '-') {
			c.offset++
		}
		start := c.offset
		for c.offset < len(c.source) && c.source[c.offset] >= '0' && c.source[c.offset] <= '9' {
			c.offset++
		}
		if c.offset == start {
			return false
		}
	}
	return true
}

func (c *jsonCursor) literal(value string) bool {
	if len(c.source)-c.offset < len(value) {
		return false
	}
	for index := range value {
		if c.source[c.offset+index] != value[index] {
			return false
		}
	}
	c.offset += len(value)
	return true
}

func (c *jsonCursor) space() {
	for c.offset < len(c.source) {
		switch c.source[c.offset] {
		case ' ', '\t', '\r', '\n':
			c.offset++
		default:
			return
		}
	}
}

func (c *jsonCursor) take(wanted byte) bool {
	if c.offset < len(c.source) && c.source[c.offset] == wanted {
		c.offset++
		return true
	}
	return false
}

func isHex(char byte) bool {
	return char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}
