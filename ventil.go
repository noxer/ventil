package ventil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// ParsingError indicates an error while parsing a KV file.
type ParsingError struct {
	File      string
	Byte      int64
	Line      int64
	PosInLine int64

	wrapped error
}

func newParsingError(err error, token token) ParsingError {
	return ParsingError{
		File:      token.file,
		Byte:      token.pos,
		Line:      token.line,
		PosInLine: token.posInLine,
		wrapped:   err,
	}
}

func (e ParsingError) Error() string {
	return fmt.Sprintf("ventil: parsing error in file %s:%d: %s", e.File, e.Line, e.wrapped)
}

func (e ParsingError) Unwrap() error {
	return e.wrapped
}

// Includer creates an including function to open included files. (Sorry)
type Includer func(name string) (io.ReadCloser, Includer, error)

// Parse reads from a reader and decodes the KV data.
func Parse(r io.Reader, includer Includer) (*KV, error) {
	return parseFile(r, includer, "<no file>")
}

// ParseFile reads from a file and decodes the KV data.
func ParseFile(name string) (*KV, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseFile(f, FileIncluder(name), name)
}

func parseFile(r io.Reader, includer Includer, filename string) (*KV, error) {
	br := newByteReader(r)

	tokens := make(chan token, 32)
	go tokenize(br, filename, tokens)

	return parse(tokens, includer)
}

func parse(tokens chan token, includer Includer) (*KV, error) {
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
		if token.typ == tokenTypeError {
			// Todo output file and line
			return nil, newParsingError(token.err, token)
		}

		if expectKey { // read the KV key
			if token.typ == tokenTypeClosingParens {
				if startKV == currentKV {
					startKV = nil
				}

				return startKV, newParsingError(errClosingParens, token)
			}

			if token.typ != tokenTypeString {
				return startKV, newParsingError(
					fmt.Errorf("ventil: Unexpected token type %s, expected String", tokenTypeNames[token.typ]),
					token,
				)
			}

			currentKV.Key = token.data
			expectKey = false
			continue
		}

		switch token.typ {
		case tokenTypeString:
			if includer != nil && (currentKV.Key == "#base" || currentKV.Key == "#include") {
				f, i, err := includer(token.data)
				if err == nil {
					innerKV, _ := parseFile(f, i, token.data)
					f.Close()

					if innerKV != nil {
						*currentKV = *innerKV
						for currentKV.NextSibling != nil {
							currentKV = currentKV.NextSibling
						}
					}
				} else {
					currentKV.HasValue = true
					currentKV.Value = token.data
				}
			} else {
				currentKV.HasValue = true
				currentKV.Value = token.data
			}
		case tokenTypeOpeningParens:
			child, err := parse(tokens, includer)
			currentKV.FirstChild = child
			if err != nil && errors.Unwrap(err) != errClosingParens {
				return startKV, err
			}
		case tokenTypeClosingParens:
			return startKV, newParsingError(errClosingParens, token)
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

func tokenize(r *reader, filename string, tokens chan token) {
	defer close(tokens)

	for {
		b, err := consumeWhitespace(r)
		if err != nil {
			if err != io.EOF {
				tokens <- token{
					file:      filename,
					pos:       r.readBytes,
					line:      r.readLines,
					posInLine: r.readBytesInLine,

					typ: tokenTypeError,
					err: err,
				}
			}
			return
		}
		currentPos := r.readBytes - 1
		currentLine := r.readLines
		currentPosInLine := r.readBytesInLine

		switch b {

		case '"': // quoted string
			str, err := readQuotedString(r)
			tokens <- token{
				file:      filename,
				pos:       currentPos,
				line:      currentLine,
				posInLine: currentPosInLine,

				typ:  tokenTypeString,
				data: str,
				err:  err,
			}
			if err != nil {
				return
			}

		case '{': // opening parenthesis
			tokens <- token{
				file:      filename,
				pos:       currentPos,
				line:      currentLine,
				posInLine: currentPosInLine,

				typ: tokenTypeOpeningParens,
			}
		case '}': // closing parenthesis
			tokens <- token{
				file:      filename,
				pos:       currentPos,
				line:      currentLine,
				posInLine: currentPosInLine,

				typ: tokenTypeClosingParens,
			}

		case '/': // comment
			line, err := r.ReadLine()
			tokens <- token{
				file:      filename,
				pos:       currentPos,
				line:      currentLine,
				posInLine: currentPosInLine,

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
				file:      filename,
				pos:       currentPos,
				line:      currentLine,
				posInLine: currentPosInLine,

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

func readUnquotedString(r *reader) (string, error) {
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

func readQuotedString(r *reader) (string, error) {
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

func consumeWhitespace(r *reader) (byte, error) {
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

// FileIncluder allows the parser to open included files.
func FileIncluder(path string, excludedFiles ...string) func(string) (io.ReadCloser, Includer, error) {
	absPath, _ := filepath.Abs(path)
	basePath := filepath.Dir(absPath)
	excludedFiles = append(excludedFiles, absPath)

	return func(name string) (io.ReadCloser, Includer, error) {
		newPath := filepath.Join(basePath, name)

		for _, excluded := range excludedFiles {
			if newPath == excluded {
				return nil, nil, fmt.Errorf("valve: include loop detected: %s", newPath)
			}
		}

		f, err := os.Open(newPath)
		if err != nil {
			return nil, nil, err
		}

		return f, FileIncluder(newPath, excludedFiles...), nil
	}
}
