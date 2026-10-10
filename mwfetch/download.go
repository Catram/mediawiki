package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Minute}

// errNotFound is a download the server does not have, which retrying will not
// fix: ExtensionDistributor names a tarball before it is built.
var errNotFound = errors.New("not found")

// retryWaits are the pauses before each retry of a download that failed on
// the network or with a server error.
var retryWaits = []time.Duration{5 * time.Second, 30 * time.Second}

// get starts a GET of url, naming the build in its User-Agent, and returns
// the response if it is a 200.
func get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return nil, fmt.Errorf("%s: %w", url, errNotFound)
		}
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return resp, nil
}

// download saves url as file and returns its SHA-256 in hex. If report is
// set, it is called as the download progresses. A failure other than
// errNotFound is retried after each of retryWaits.
func download(url, file string, report func(done, total int64)) (string, error) {
	for attempt := 0; ; attempt++ {
		sum, err := downloadOnce(url, file, report)
		if err == nil || errors.Is(err, errNotFound) || attempt == len(retryWaits) {
			return sum, err
		}
		fmt.Fprintf(os.Stderr, "%s: %v; trying again in %v\n", path.Base(url), err, retryWaits[attempt])
		time.Sleep(retryWaits[attempt])
	}
}

func downloadOnce(url, file string, report func(done, total int64)) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	var w io.Writer = io.MultiWriter(f, h)
	if report != nil {
		w = &progress{w: w, total: resp.ContentLength, report: report}
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// progress counts what passes through it and calls report every step bytes.
type progress struct {
	w      io.Writer
	done   int64
	total  int64 // -1 if unknown
	report func(done, total int64)
}

const step = 32 << 20

func (p *progress) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if (p.done+int64(n))/step > p.done/step {
		p.report(p.done+int64(n), p.total)
	}
	p.done += int64(n)
	return n, err
}
