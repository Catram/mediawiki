package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// pin is a row of extensions.tsv: a skin or extension and the
// ExtensionDistributor tarball it is pinned to.
type pin struct {
	Path   string // skins/<name> or extensions/<name>
	URL    string // empty until pinned
	SHA256 string // of the tarball, in hex
	Note   string
}

var pinPath = regexp.MustCompile(`^(skins|extensions)/([A-Za-z0-9][A-Za-z0-9_.-]*)$`)

// Kind is "skins" or "extensions".
func (p pin) Kind() string { return pinPath.FindStringSubmatch(p.Path)[1] }

// Name is the skin's or extension's name, and the folder its tarball holds.
func (p pin) Name() string { return pinPath.FindStringSubmatch(p.Path)[2] }

// pinList is extensions.tsv: a header, then a row per pin. Blank lines are
// dropped.
type pinList struct {
	Header string
	Pins   []pin
}

func readPins(file string) (*pinList, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	l := &pinList{Header: lines[0]}
	seen := map[string]bool{}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) > 4 {
			return nil, fmt.Errorf("%s:%d: more than 4 fields", file, i+2)
		}
		fields = append(fields, make([]string, 4-len(fields))...)
		p := pin{Path: fields[0], URL: fields[1], SHA256: fields[2], Note: fields[3]}
		if !pinPath.MatchString(p.Path) {
			return nil, fmt.Errorf("%s:%d: %q is not skins/<name> or extensions/<name>", file, i+2, p.Path)
		}
		if seen[p.Path] {
			return nil, fmt.Errorf("%s:%d: %s is listed twice", file, i+2, p.Path)
		}
		seen[p.Path] = true
		if (p.URL == "") != (p.SHA256 == "") {
			return nil, fmt.Errorf("%s:%d: %s has a URL or a SHA-256 but not both", file, i+2, p.Path)
		}
		if p.SHA256 != "" && !isSHA256(p.SHA256) {
			return nil, fmt.Errorf("%s:%d: %s: %q is not a SHA-256", file, i+2, p.Path, p.SHA256)
		}
		l.Pins = append(l.Pins, p)
	}
	return l, nil
}

// write replaces file with the list. The file keeps its mode.
func (l *pinList) write(file string) error {
	var b strings.Builder
	b.WriteString(l.Header + "\n")
	for _, p := range l.Pins {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", p.Path, p.URL, p.SHA256, p.Note)
	}
	return os.WriteFile(file, []byte(b.String()), 0o644)
}

func isSHA256(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && s == strings.ToLower(s)
}
