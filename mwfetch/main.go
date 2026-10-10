// Command mwfetch downloads MediaWiki and the skins and extensions pinned in
// extensions.tsv, and moves those pins. The image build runs it in a stage of
// its own and copies the result; the weekly build runs it to move the pins.
//
//	mwfetch install -version <version> -sha256 <SHA-256> [-list extensions.tsv] <web root>
//	mwfetch update [-containerfile Containerfile] [-list extensions.tsv]
//
// install puts the release, with vendor/ and its bundled skins and extensions
// but without the web installer, in the web root, which must not exist yet,
// and adds each pinned tarball. Every download is checked against its
// SHA-256.
//
// update asks ExtensionDistributor for the current tarball of each row for the
// release's branch (REL1_46 for MediaWiki 1.46), downloads those that moved to
// take their SHA-256, and rewrites the list. It prints a line for each pin it
// moved, "<path>: <old tarball> -> <new tarball>", and nothing else on
// standard output. A row with only a path is pinned for the first time. If a
// tarball cannot be downloaded yet, nothing is changed.
package main

import (
	"fmt"
	"log"
	"os"
)

// userAgent names the build, as Wikimedia's User-Agent policy asks.
const userAgent = "Catram-mediawiki-build (https://github.com/Catram/mediawiki)"

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  mwfetch install -version <version> -sha256 <SHA-256> [-list extensions.tsv] <web root>
  mwfetch update [-containerfile Containerfile] [-list extensions.tsv]
`)
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("mwfetch: ")
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "install":
		err = runInstall(os.Args[2:])
	case "update":
		err = runUpdate(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}
