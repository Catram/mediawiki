#!/bin/sh
# Installs MediaWiki and the listed skins and extensions into the web root.
# Run by the Containerfile, which provides wget.
#
#   install.sh <MediaWiki version> <SHA-256 of its tarball> <web root>
#
# Core comes from releases.wikimedia.org, with vendor/ and the bundled skins
# and extensions. The others are ExtensionDistributor's tarballs pinned in
# extensions.tsv next to this script; update-extensions.sh moves the pins.
# Every download is checked against its SHA-256.
set -eu

version=$1
core_sha256=$2
root=${3:?web root}
here=$(dirname "$0")
tab=$(printf '\t')
work=$(mktemp -d)

# fail MESSAGE: stops the install, saying why.
fail() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

# rows FILE: the rows of a TSV file, without its header and blank lines.
rows() {
	tail -n +2 "$1" | grep -v '^[[:space:]]*$' || true
}

# Core, without the web installer. Its tarball is large, so wget shows its
# progress: a line for every 32 MB.
printf '%s\n' "MediaWiki $version"
wget --progress=dot:giga -O "$work/core.tar.gz" \
	"https://releases.wikimedia.org/mediawiki/${version%.*}/mediawiki-$version.tar.gz"
echo "$core_sha256  $work/core.tar.gz" | sha256sum -c - ||
	fail "MediaWiki $version does not match its SHA-256"
tar -xzf "$work/core.tar.gz" -C "$work"
rm -rf "$root"
mv "$work/mediawiki-$version" "$root"
rm -rf "$root/mw-config"

# The rest: each path is skins/<name> or extensions/<name>, and its tarball
# holds one folder, <name>. One that the release already bundles is an error,
# not an overwrite.
rows "$here/extensions.tsv" | while IFS=$tab read -r path url sha256 _; do
	name=${path##*/}
	printf '%s\n' "$path: ${url##*/}"
	if [ -e "$root/$path" ]; then
		fail "$path: already bundled with MediaWiki $version; remove it from extensions.tsv"
	fi
	wget -nv -O "$work/download.tar.gz" "$url" ||
		fail "$path: cannot download ${url##*/}; run update-extensions.sh to move the pin"
	echo "$sha256  $work/download.tar.gz" | sha256sum -c - ||
		fail "$path: ${url##*/} does not match its pinned SHA-256"
	rm -rf "$work/unpacked"
	mkdir "$work/unpacked"
	tar -xzf "$work/download.tar.gz" -C "$work/unpacked"
	if [ ! -d "$work/unpacked/$name" ]; then
		fail "$path: ${url##*/} does not hold a folder named $name"
	fi
	mv "$work/unpacked/$name" "$root/$path"
done

rm -rf "$work"
printf '%s\n' "Installed MediaWiki $version in $root, with $(rows "$here/extensions.tsv" | wc -l | tr -d ' ') skins and extensions from extensions.tsv"
