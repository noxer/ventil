package ventil

import (
	"bufio"
	"io"
)

// reader wraps a bufio.Reader and counts bytes.
type reader struct {
	r                  *bufio.Reader
	lastByteWasNewline bool
	readBytes          int64
	readLines          int64
	oldReadBytesInLine int64
	readBytesInLine    int64
}

func newByteReader(r io.Reader) *reader {
	return &reader{
		r:         bufio.NewReader(r),
		readLines: 1,
	}
}

// ReadByte reads a single byte from the input.
func (r *reader) ReadByte() (byte, error) {
	b, err := r.r.ReadByte()
	if err != nil {
		return b, err
	}

	r.readBytes++

	if b == '\n' {
		r.readLines++
		r.oldReadBytesInLine = r.readBytesInLine
		r.readBytesInLine = 0
		r.lastByteWasNewline = true
	}

	return b, err
}

// UnreadByte returns a byte to the buffer.
func (r *reader) UnreadByte() error {
	err := r.r.UnreadByte()
	if err != nil {
		return err
	}

	r.readBytes--

	if r.lastByteWasNewline {
		r.readLines--
		r.readBytesInLine = r.oldReadBytesInLine
		r.lastByteWasNewline = false
	}

	return err
}

func (r *reader) ReadLine() ([]byte, error) {
	line, err := r.r.ReadBytes('\n')

	r.readBytes += int64(len(line))

	r.lastByteWasNewline = false
	if len(line) > 0 && line[len(line)-1] == '\n' {
		r.readLines++
		r.oldReadBytesInLine = r.readBytesInLine
		r.readBytesInLine = 0
		r.lastByteWasNewline = true

		line = line[:len(line)-1]
	}

	return line, err
}
