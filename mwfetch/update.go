package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// argVersion finds the release in the Containerfile.
var argVersion = regexp.MustCompile(`(?m)^ARG MW_VERSION=(\S+)$`)

func runUpdate(args []string) error {
	flags := flag.NewFlagSet("update", flag.ExitOnError)
	flags.Usage = usage
	containerfile := flags.String("containerfile", "Containerfile", "the Containerfile that sets MW_VERSION")
	list := flags.String("list", "extensions.tsv", "the pinned skins and extensions")
	flags.Parse(args)
	if flags.NArg() != 0 {
		usage()
	}

	data, err := os.ReadFile(*containerfile)
	if err != nil {
		return err
	}
	a := argVersion.FindSubmatch(data)
	if a == nil {
		return fmt.Errorf("%s: no ARG MW_VERSION=", *containerfile)
	}
	m := mwVersion.FindStringSubmatch(string(a[1]))
	if m == nil {
		return fmt.Errorf("%s: %q is not a MediaWiki release such as 1.46.2", *containerfile, a[1])
	}
	branch := "REL" + strings.ReplaceAll(m[1], ".", "_")

	pins, err := readPins(*list)
	if err != nil {
		return err
	}
	if len(pins.Pins) == 0 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Asking ExtensionDistributor for the %s tarballs\n", branch)
	current, err := tarballs(pins.Pins, branch)
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "mwfetch-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	var moved []string
	for i, p := range pins.Pins {
		now := current[p.Path]
		if now == p.URL {
			continue
		}
		sum, err := download(now, filepath.Join(work, "download.tar.gz"), nil)
		if errors.Is(err, errNotFound) {
			return fmt.Errorf("%s: %s is not built yet; nothing is changed, try again later", p.Path, path.Base(now))
		}
		if err != nil {
			return fmt.Errorf("%s: %w", p.Path, err)
		}
		was := "(not pinned)"
		if p.URL != "" {
			was = path.Base(p.URL)
		}
		moved = append(moved, fmt.Sprintf("%s: %s -> %s", p.Path, was, path.Base(now)))
		pins.Pins[i].URL, pins.Pins[i].SHA256 = now, sum
	}
	if len(moved) == 0 {
		fmt.Fprintln(os.Stderr, "Every pin is current")
		return nil
	}
	if err := pins.write(*list); err != nil {
		return err
	}
	for _, line := range moved {
		fmt.Println(line)
	}
	return nil
}

// tarballs asks ExtensionDistributor's API, in one request, for the current
// tarball for branch of each pin, and returns their URLs by path. Wikimedia
// limits how often a client may ask.
func tarballs(pins []pin, branch string) (map[string]string, error) {
	names := map[string][]string{}
	for _, p := range pins {
		names[p.Kind()] = append(names[p.Kind()], p.Name())
	}
	q := url.Values{
		"action":        {"query"},
		"list":          {"extdistbranches"},
		"format":        {"json"},
		"formatversion": {"2"},
	}
	if n := names["extensions"]; len(n) > 0 {
		q.Set("edbexts", strings.Join(n, "|"))
	}
	if n := names["skins"]; len(n) > 0 {
		q.Set("edbskins", strings.Join(n, "|"))
	}
	resp, err := get("https://www.mediawiki.org/w/api.php?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var answer struct {
		Error *struct {
			Code string `json:"code"`
			Info string `json:"info"`
		} `json:"error"`
		Warnings json.RawMessage `json:"warnings"`
		Query    struct {
			// kind, name, branch: URL. A name also has "source", its Git
			// repository.
			Branches map[string]map[string]map[string]string `json:"extdistbranches"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("ExtensionDistributor's answer: %w", err)
	}
	if answer.Error != nil {
		return nil, fmt.Errorf("ExtensionDistributor: %s: %s", answer.Error.Code, answer.Error.Info)
	}
	if len(answer.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "ExtensionDistributor warns: %s\n", answer.Warnings)
	}
	urls := map[string]string{}
	for _, p := range pins {
		u := answer.Query.Branches[p.Kind()][p.Name()][branch]
		if u == "" {
			return nil, fmt.Errorf("%s: no %s tarball on ExtensionDistributor", p.Path, branch)
		}
		urls[p.Path] = u
	}
	return urls, nil
}
