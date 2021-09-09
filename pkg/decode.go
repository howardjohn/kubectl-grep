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
	separator     = "---"
	listSeparator = "- "
)

type Reader interface {
	Read() ([]byte, error)
}

type YAMLReader struct {
	reader Reader
	isList bool
}

func NewYAMLReader(r *bufio.Reader, isList bool) *YAMLReader {
	return &YAMLReader{
		reader: &LineReader{reader: r},
		isList: isList,
	}
}

// Read returns a full YAML document.
func (r *YAMLReader) Read() ([]byte, error) {
	var buffer bytes.Buffer
	for {
		line, err := r.reader.Read()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if r.isList {
			// TODO: optimize if we know its a list
			//sep := len([]byte(listSeparator))
			//if i := bytes.Index(line, []byte(listSeparator)); i == 0 {
			//	fmt.Println(string(line))
			//	i+=sep
			//}
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
