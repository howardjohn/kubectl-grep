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

// Read returns a full YAML document.
func (r *YAMLReader) Read() ([]byte, error) {
	var buffer bytes.Buffer
	listDiscard := make([]byte, 2, 2)
	firstLoop := true
	for {
		if r.isList {
			if !firstLoop {
				rr, err := r.reader.reader.Peek(1)
				if err != nil {
					return nil, err
				}
				if rr[0] == '-' {
					// We hit the next entry
					return buffer.Bytes(), nil
				}
				if rr[0] != ' ' {
					// Not part of the list anymore, just be end of the list
					// Drain the list so we don't read more
					if _, err := io.Copy(io.Discard, r.reader.reader); err != nil {
						return nil, err
					}
					return buffer.Bytes(), nil
				}
			}
			_, err := io.ReadFull(r.reader.reader, listDiscard)
			if err != nil {
				return nil, err
			}
			firstLoop = false
		}

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
	reader *bufio.Reader
}

func (r *LineReader) Peak(n int) ([]byte, error) {
	return r.reader.Peek(n)
}

// Read returns a single line (with '\n' ended) from the underlying reader.
// An error is returned iff there is an error with the underlying reader.
func (r *LineReader) Read() ([]byte, error) {
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
