package pkg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Implementation borrowed from k8s.io/apimachinery to avoid large imports

const (
	separator = "---"
)

type YAMLReader struct {
	reader *LineReader
	isList bool
}

var listFastHeader = `apiVersion: v1
items:
-`

var listFastHeaderTrim = `apiVersion: v1
items:
`

func NewYAMLReader(r *bufio.Reader) *YAMLReader {
	b, _ := r.Peek(len(listFastHeader))
	isList := string(b) == listFastHeader
	reader := &LineReader{reader: r}
	if isList {
		if _, err := reader.reader.Discard(len(listFastHeaderTrim)); err != nil {
			return nil
		}
	}
	return &YAMLReader{
		reader: reader,
		isList: isList,
	}
}

func (r *YAMLReader) Read() ([]byte, error) {
	if !r.isList {
		return r.readFlat()
	}
	firstLine := true
	var buffer bytes.Buffer
	for {
		start := r.reader.StartsList()
		if start && !firstLine {
			return buffer.Bytes(), nil
		}
		firstLine = false
		l, err := r.reader.Read()
		if err != nil {
			return nil, err
		}
		switch len(l) {
		case 0: // can have empty lines from YAMLs with newline in string.
			return nil, fmt.Errorf("invalid line: %q", string(l))
		case 1:
			if l[0] != '\n' {
				// Should not happen
				return nil, fmt.Errorf("invalid line: %q", string(l))
			}
		default: // Trim the start
			if l[0] == 'k' && l[1] == 'i' {
				if _, err := io.Copy(io.Discard, r.reader.reader); err != nil {
					return nil, err
				}
				// End of list. TODO: more robust check
				return buffer.Bytes(), err
			}
			l = l[2:]
		}
		buffer.Write(l)
	}
}

func (r *YAMLReader) readFlat() ([]byte, error) {
	var buffer bytes.Buffer
	for {
		line, err := r.reader.Read()
		if err != nil && err != io.EOF {
			return nil, err
		}

		sep := len([]byte(separator))
		if i := bytes.Index(line, []byte(separator)); i == 0 {
			// We have a potential document terminator
			i += sep
			trimmed := strings.TrimSpace(string(line[i:]))
			// We only allow comments and spaces following the yaml doc separator, otherwise we'll return an error
			if len(trimmed) > 0 && string(trimmed[0]) != "#" {
				return nil, fmt.Errorf("invalid Yaml document separator: %s", trimmed)
			}
			if buffer.Len() != 0 {
				return buffer.Bytes(), nil
			}
			if err == io.EOF {
				return nil, err
			}
		}
		if err == io.EOF {
			if buffer.Len() != 0 {
				// If we're at EOF, we have a final, non-terminated line. Return it.
				return buffer.Bytes(), nil
			}
			return nil, err
		}
		buffer.Write(line)
	}
}

type LineReader struct {
	reader   *bufio.Reader
	nextLine []byte
}

func (r *LineReader) Peak(n int) ([]byte, error) {
	return r.reader.Peek(n)
}

// StartsList checks if the next line starts a list
func (r *LineReader) StartsList() bool {
	if r.nextLine != nil {
		return r.nextLine[0] == '-' && r.nextLine[1] == ' '
	}
	l, err := r.Read()
	if err != nil {
		return false
	}
	r.nextLine = l
	return r.nextLine[0] == '-' && r.nextLine[1] == ' '
}

func (r *LineReader) Read() ([]byte, error) {
	if r.nextLine != nil {
		res := r.nextLine
		r.nextLine = nil
		return res, nil
	}
	var (
		isPrefix bool  = true
		err      error = nil
		line     []byte
		buffer   bytes.Buffer
	)

	for isPrefix && err == nil {
		line, isPrefix, err = r.reader.ReadLine()
		buffer.Write(line)
	}
	buffer.WriteByte('\n')
	return buffer.Bytes(), err
}
