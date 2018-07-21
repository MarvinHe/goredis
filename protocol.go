package go_redis

import (
	"fmt"
	// "strconv"
)

var HEX2DIGIT map[byte]byte = map[byte]byte {
	'0': 0,
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9,
	'a': 10,
	'A': 10,
	'b': 11,
	'B': 11,
	'c': 12,
	'C': 12,
	'd': 13,
	'D': 13,
	'e': 14,
	'E': 14,
	'f': 15,
	'F': 15,
}

func ParseLine(line []byte) []string {
	var inq, insq, done bool
	var buf []byte = []byte{}
	ret := []string{}
	var i, l int
	var b, b1 byte
	l = len(line)
	for i < l {
		inq, insq, done = false, false, false
		for !done && i < l {
			b = line[i]
			if inq {
				if b == '\\' && line[i+1] == 'x' {
					buf = append(buf, HEX2DIGIT[line[i+2]] * 16 + HEX2DIGIT[line[i+3]])
					i += 3
				} else if b == '\\' {
					switch line[i+1] {
					case 'n':
						b1 = '\n'
					case 'r':
						b1 = '\r'
					case 't':
						b1 = '\t'
					case 'b':
						b1 = '\b'
					case 'a':
						b1= '\a'
					default:
						b1 = line[i+1]
					}
					buf = append(buf, b1)
					i += 1
				} else if b == '"' {
					done = true
				} else {
					buf = append(buf, b)
				}
			} else if insq {
				if b == '\\' && line[i+1] == '\'' {
					i += 1
					buf = append(buf, '\'')
				} else if b == '\'' {
					done = true
				} else {
					buf = append(buf, b)
				}
			} else {
				switch b {
				case ' ', '\r', '\n', '\t', 0:
					done = true
				case '"':
					inq = true
				case '\'':
					insq = true
				default:
					buf = append(buf, b)
				}
			}
			i += 1
		}

		ret = append(ret, string(buf))
		buf = buf[:0]
	}
	return ret
}

func EncodeReply(data interface{}) string {
	// var sb []byte = []byte{}
	switch data.(type) {
	case string:
		// TODO: test Atoi and Sprintf, which is faster when converting %d
		return fmt.Sprintf("$%d%s%s%s", len(data.(string)), LINE_DELIMITER, data.(string), LINE_DELIMITER)
	case int64:
		return fmt.Sprintf(":%d%s", data.(int64), LINE_DELIMITER)
	}
	return REPLAY_NULL_BULK
}

func ErrorReply(s string) string {
	return fmt.Sprintf("%s %s%s", ERR, s, LINE_DELIMITER)
}