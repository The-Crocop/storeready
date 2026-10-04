package scanner

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"unicode/utf16"
)

func readZipEntry(file *zip.File, max int64) ([]byte, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, max))
}

func containsText(data []byte, value string) bool {
	if bytes.Contains(data, []byte(value)) {
		return true
	}
	runes := utf16.Encode([]rune(value))
	le := make([]byte, 0, len(runes)*2)
	be := make([]byte, 0, len(runes)*2)
	for _, r := range runes {
		le = append(le, byte(r), byte(r>>8))
		be = append(be, byte(r>>8), byte(r))
	}
	return bytes.Contains(data, le) || bytes.Contains(data, be)
}

func hasPath(files []*zip.File, predicate func(string) bool) bool {
	for _, f := range files {
		if predicate(strings.ReplaceAll(f.Name, "\\", "/")) {
			return true
		}
	}
	return false
}
