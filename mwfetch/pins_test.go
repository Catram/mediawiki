package main

import (
	"os"
	"path/filepath"
	"testing"
)

const sum = "0022bd1a9f8047e9bf7294acb79e025368842fcf7d395757632f8ace31310868"

func TestPinsRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "extensions.tsv")
	in := "path\turl\tsha256\tnote\n" +
		"skins/Modern\thttps://example.org/Modern.tar.gz\t" + sum + "\tkept\n" +
		"\n" +
		"extensions/New\n"
	if err := os.WriteFile(file, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := readPins(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Pins) != 2 || l.Pins[0].Note != "kept" || l.Pins[1].URL != "" {
		t.Fatalf("read %+v", l.Pins)
	}
	if k, n := l.Pins[1].Kind(), l.Pins[1].Name(); k != "extensions" || n != "New" {
		t.Errorf("kind %q, name %q", k, n)
	}
	if err := l.write(file); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(file)
	want := "path\turl\tsha256\tnote\n" +
		"skins/Modern\thttps://example.org/Modern.tar.gz\t" + sum + "\tkept\n" +
		"extensions/New\t\t\t\n"
	if string(out) != want {
		t.Errorf("wrote %q, want %q", out, want)
	}
}

func TestPinsRefused(t *testing.T) {
	for name, row := range map[string]string{
		"bad kind":     "vendor/x",
		"escape":       "extensions/..",
		"nested":       "extensions/a/b",
		"url only":     "extensions/A\thttps://example.org/A.tar.gz",
		"bad sum":      "extensions/A\thttps://example.org/A.tar.gz\tabc",
		"listed twice": "extensions/A\nextensions/A",
	} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "extensions.tsv")
			if err := os.WriteFile(file, []byte("path\turl\tsha256\tnote\n"+row+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := readPins(file); err == nil {
				t.Error("accepted")
			}
		})
	}
}
