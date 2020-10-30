package ventil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Token type constants
const (
	tokenTypeError         = 0
	tokenTypeString        = 1
	tokenTypeOpeningParens = 2
	tokenTypeClosingParens = 3
	tokenTypeComment       = 4
)

type token struct {
	file      string
	pos       int64
	line      int64
	posInLine int64

	typ  int
	data string
	err  error
}

var tokenTypeNames = map[int]string{
	tokenTypeError:         "Error",
	tokenTypeString:        "String",
	tokenTypeOpeningParens: "{",
	tokenTypeClosingParens: "}",
	tokenTypeComment:       "Comment",
}

var errClosingParens = errors.New("ventil: unexpected closing parenthesis")

// Parse reads from a reader and decodes the KV data.
func Parse(r io.Reader) (*KV, error) {
	br := bufio.NewReader(r)

	tokens := make(chan token, 32)
	go tokenize(br, tokens)

	return parse(tokens)
}

// ParseFile reads from a file and decodes the KV data.
func ParseFile(name string) (*KV, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

func parse(tokens chan token) (*KV, error) {
	var (
		startKV   = &KV{}
		prevKV    *KV
		currentKV = startKV
	)

	expectKey := true
	for token := range tokens {
		if token.typ == tokenTypeComment {
			continue
		}

		if expectKey { // read the KV key
			if token.typ == tokenTypeClosingParens {
				break
			}

			if token.typ != tokenTypeString {
				return startKV, fmt.Errorf("ventil: Unexpected token type %s, expected String", tokenTypeNames[token.typ])
			}

			currentKV.Key = token.data
			expectKey = false
			continue
		}

		switch token.typ {
		case tokenTypeString:
			currentKV.HasValue = true
			currentKV.Value = token.data
		case tokenTypeOpeningParens:
			child, err := parse(tokens)
			currentKV.FirstChild = child
			if err != nil && err != errClosingParens {
				return startKV, err
			}
		case tokenTypeClosingParens:
			return startKV, errClosingParens
		}

		nextKV := &KV{}
		currentKV.NextSibling = nextKV
		prevKV = currentKV
		currentKV = nextKV

		expectKey = true
	}

	if prevKV != nil {
		prevKV.NextSibling = nil
	}
	if startKV == currentKV {
		startKV = nil
	}

	return startKV, nil
}

func (t token) String() string {
	typeNames := map[int]string{
		tokenTypeError:         "Error",
		tokenTypeString:        "String",
		tokenTypeOpeningParens: "{",
		tokenTypeClosingParens: "}",
		tokenTypeComment:       "Comment",
	}

	return fmt.Sprintf("Token: %s \"%s\" Error: %s", typeNames[t.typ], t.data, t.err)
}

func tokenize(r *bufio.Reader, tokens chan token) {
	defer close(tokens)

	for {
		b, err := consumeWhitespace(r)
		if err != nil {
			if err != io.EOF {
				tokens <- token{
					typ: tokenTypeError,
					err: err,
				}
			}
			return
		}

		switch b {

		case '"': // quoted string
			str, err := readQuotedString(r)
			tokens <- token{
				typ:  tokenTypeString,
				data: str,
				err:  err,
			}
			if err != nil {
				return
			}

		case '{': // opening parenthesis
			tokens <- token{typ: tokenTypeOpeningParens}
		case '}': // closing parenthesis
			tokens <- token{typ: tokenTypeClosingParens}

		case '/': // comment
			line, _, err := r.ReadLine()
			tokens <- token{
				typ:  tokenTypeComment,
				data: string(line),
				err:  err,
			}
			if err != nil {
				return
			}

		default:
			r.UnreadByte()
			str, err := readUnquotedString(r)
			tokens <- token{
				typ:  tokenTypeString,
				data: str,
				err:  err,
			}
			if err != nil {
				return
			}
		}
	}
}

func readUnquotedString(r *bufio.Reader) (string, error) {
	var buf strings.Builder
	escaped := false

	for {
		b, err := r.ReadByte()
		if err != nil {
			return buf.String(), err
		}

		if escaped {
			escaped = false

			switch b {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			default:
				buf.WriteByte(b)
			}

			continue
		}

		if b == '\\' {
			escaped = true
			continue
		}

		if isWhitespace(b) || b == '{' || b == '}' || b == '/' || b == '"' {
			r.UnreadByte()
			return buf.String(), nil
		}

		buf.WriteByte(b)
	}

}

func readQuotedString(r *bufio.Reader) (string, error) {
	var buf strings.Builder
	escaped := false

	for {
		b, err := r.ReadByte()
		if err != nil {
			return buf.String(), err
		}

		if escaped {
			escaped = false

			switch b {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			default:
				buf.WriteByte(b)
			}

			continue
		}

		if b == '\\' {
			escaped = true
			continue
		}

		if b == '"' {
			return buf.String(), nil
		}

		buf.WriteByte(b)
	}
}

func consumeWhitespace(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		if !isWhitespace(b) {
			return b, nil
		}
	}
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
