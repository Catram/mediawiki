package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

// mwVersion is a release such as 1.46.2; the first group is its series, 1.46.
var mwVersion = regexp.MustCompile(`^([0-9]+\.[0-9]+)\.[0-9]+$`)

func runInstall(args []string) error {
	flags := flag.NewFlagSet("install", flag.ExitOnError)
	flags.Usage = usage
	version := flags.String("version", "", "MediaWiki version, e.g. 1.46.2")
	coreSHA256 := flags.String("sha256", "", "SHA-256 of the release tarball")
	list := flags.String("list", "extensions.tsv", "the pinned skins and extensions")
	flags.Parse(args)
	if flags.NArg() != 1 || *version == "" || *coreSHA256 == "" {
		usage()
	}
	root := flags.Arg(0)
	m := mwVersion.FindStringSubmatch(*version)
	if m == nil {
		return fmt.Errorf("%q is not a MediaWiki release such as 1.46.2", *version)
	}
	if !isSHA256(*coreSHA256) {
		return fmt.Errorf("%q is not a SHA-256", *coreSHA256)
	}
	if _, err := os.Lstat(root); !errors.Is(err, fs.ErrNotExist) {
		if err == nil {
			err = errors.New("already exists")
		}
		return fmt.Errorf("%s: %w", root, err)
	}

	// Every row must be pinned before anything is downloaded.
	pins, err := readPins(*list)
	if err != nil {
		return err
	}
	for _, p := range pins.Pins {
		if p.URL == "" {
			return fmt.Errorf("%s: not pinned yet; run mwfetch update", p.Path)
		}
	}

	// Next to the web root, so what is unpacked can be moved into it.
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return err
	}
	work, err := os.MkdirTemp(filepath.Dir(root), ".mwfetch-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	// Core, without the web installer. Its tarball is large, so the download
	// shows its progress.
	fmt.Printf("MediaWiki %s\n", *version)
	url := fmt.Sprintf("https://releases.wikimedia.org/mediawiki/%s/mediawiki-%s.tar.gz", m[1], *version)
	report := func(done, total int64) {
		if total > 0 {
			fmt.Printf("  %d of %d MB\n", done>>20, total>>20)
		} else {
			fmt.Printf("  %d MB\n", done>>20)
		}
	}
	if err := fetch(url, *coreSHA256, work, "mediawiki-"+*version, root, report); err != nil {
		return fmt.Errorf("MediaWiki %s: %w", *version, err)
	}
	if err := os.RemoveAll(filepath.Join(root, "mw-config")); err != nil {
		return err
	}

	// The rest: each tarball holds one folder, named after the skin or
	// extension. One that the release already bundles is an error, not an
	// overwrite.
	for _, p := range pins.Pins {
		fmt.Printf("%s: %s\n", p.Path, path.Base(p.URL))
		target := filepath.Join(root, filepath.FromSlash(p.Path))
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("%s: already bundled with MediaWiki %s; remove it from %s", p.Path, *version, *list)
		}
		if err := fetch(p.URL, p.SHA256, work, p.Name(), target, nil); err != nil {
			if errors.Is(err, errNotFound) {
				err = fmt.Errorf("%w; run mwfetch update to move the pin", err)
			}
			return fmt.Errorf("%s: %w", p.Path, err)
		}
	}

	fmt.Printf("Installed MediaWiki %s in %s, with %d skins and extensions from %s\n",
		*version, root, len(pins.Pins), filepath.Base(*list))
	return nil
}

// fetch downloads the tarball at url into work, checks it against want,
// unpacks it there, and moves the folder in it named folder to target.
func fetch(url, want, work, folder, target string, report func(done, total int64)) error {
	tarball := filepath.Join(work, "download.tar.gz")
	got, err := download(url, tarball, report)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s does not match its pinned SHA-256: it is %s", path.Base(url), got)
	}
	unpacked := filepath.Join(work, "unpacked")
	if err := os.RemoveAll(unpacked); err != nil {
		return err
	}
	if err := os.Mkdir(unpacked, 0o755); err != nil {
		return err
	}
	if err := extract(tarball, unpacked); err != nil {
		return err
	}
	src := filepath.Join(unpacked, folder)
	if fi, err := os.Lstat(src); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s does not hold a folder named %s", path.Base(url), folder)
	}
	if err := os.Chmod(src, 0o755); err != nil {
		return err
	}
	return os.Rename(src, target)
}
