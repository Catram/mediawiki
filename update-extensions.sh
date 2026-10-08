#!/bin/sh
# Moves the pins in extensions.tsv to ExtensionDistributor's current tarballs
# for the release's branch (REL1_46 for MediaWiki 1.46), and prints what moved.
# A row with only a path is pinned for the first time. Run by the weekly build,
# which commits the result; run it by hand after adding a row or changing
# MW_VERSION. Needs curl and Python.
#
#   sh update-extensions.sh
#
# ExtensionDistributor's API can name a tarball before it is built. Then the
# download fails, nothing is changed, and the next run tries again.
set -eu

here=$(dirname "$0")
tab=$(printf '\t')
version=$(sed -n 's/^ARG MW_VERSION=//p' "$here/Containerfile")
test -n "$version"
branch=REL$(echo "${version%.*}" | tr . _)
work=$(mktemp -d "${TMPDIR:-/tmp}/extensions.XXXXXX")

# rows FILE: the rows of a TSV file, without its header and blank lines.
rows() {
	tail -n +2 "$1" | grep -v '^[[:space:]]*$' || true
}

# names KIND: the names of the skins or extensions in extensions.tsv,
# separated by |.
names() {
	rows "$here/extensions.tsv" | cut -f 1 | grep "^$1/" | cut -d / -f 2 | paste -s -d '|' -
}

# tarball PATH: the URL of the current tarball for $branch of PATH, e.g.
# extensions/Disambiguator, in $work/branches.json. Fails if there is none.
tarball() {
	python3 -c '
import json, sys
path, branch, answer = sys.argv[1:]
kind, name = path.split("/")
with open(answer) as f:
    branches = json.load(f)["query"]["extdistbranches"].get(kind, {}).get(name, {})
if branch not in branches:
    sys.exit(f"update-extensions.sh: {path}: no {branch} tarball on ExtensionDistributor")
print(branches[branch])
' "$1" "$branch" "$work/branches.json"
}

# sha256 FILE: the file's SHA-256, on Linux or macOS.
sha256() {
	if command -v sha256sum > /dev/null; then
		sha256sum "$1"
	else
		shasum -a 256 "$1"
	fi | cut -d ' ' -f 1
}

# All in one request: Wikimedia limits how often a client may ask, and its
# User-Agent policy asks for a name.
printf '%s\n' "Asking ExtensionDistributor for the $branch tarballs"
curl -fsS -o "$work/branches.json" \
	--user-agent "Catram-mediawiki-build (https://github.com/Catram/mediawiki)" \
	"https://www.mediawiki.org/w/api.php?action=query&list=extdistbranches&format=json&formatversion=2&edbexts=$(names extensions)&edbskins=$(names skins)"

head -n 1 "$here/extensions.tsv" > "$work/extensions.tsv"
rows "$here/extensions.tsv" | while IFS=$tab read -r path url sum note; do
	current=$(tarball "$path")
	if [ "$current" != "$url" ]; then
		printf '%s\n' "$path: ${url##*/} -> ${current##*/}" >&2
		if ! curl -fsS -o "$work/download.tar.gz" "$current"; then
			printf '%s\n' "update-extensions.sh: $path: ${current##*/} is not built yet; try again later" >&2
			exit 1
		fi
		url=$current
		sum=$(sha256 "$work/download.tar.gz")
	fi
	printf '%s\t%s\t%s\t%s\n' "$path" "$url" "$sum" "$note"
done >> "$work/extensions.tsv"

# Over the old file, so it keeps its mode.
cat "$work/extensions.tsv" > "$here/extensions.tsv"
rm -rf "$work"
