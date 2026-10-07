#!/bin/sh
# Installs MediaWiki and the pinned skins and extensions into the web root.
# Run by the Containerfile, which provides wget.
#
#   install.sh <MediaWiki version> <SHA-256 of its tarball> <web root>
#
# Core comes from releases.wikimedia.org, with vendor/ and the bundled skins
# and extensions. The others come from ExtensionDistributor, as pinned in
# extensions.tsv next to this script. Every download is checked against its
# SHA-256.
set -eu

version=$1
core_sha256=$2
root=${3:?web root}
here=$(dirname "$0")
tab=$(printf '\t')
work=$(mktemp -d)

# fetch URL SHA256: download URL, check it, and unpack it into $work/unpacked.
fetch() {
	wget -nv -O "$work/download.tar.gz" "$1"
	echo "$2  $work/download.tar.gz" | sha256sum -c -
	rm -rf "$work/unpacked"
	mkdir "$work/unpacked"
	tar -xzf "$work/download.tar.gz" -C "$work/unpacked"
}

# rows FILE: the rows of a TSV file, without its header and blank lines.
rows() {
	tail -n +2 "$1" | grep -v '^[[:space:]]*$' || true
}

# Core, without the web installer.
fetch "https://releases.wikimedia.org/mediawiki/${version%.*}/mediawiki-$version.tar.gz" "$core_sha256"
rm -rf "$root"
mv "$work/unpacked/mediawiki-$version" "$root"
rm -rf "$root/mw-config"

# The rest: each tarball holds one folder, named like the last part of the
# path. One that the release already bundles is an error, not an overwrite.
rows "$here/extensions.tsv" | while IFS=$tab read -r path url sha256 _; do
	echo "$path"
	fetch "$url" "$sha256"
	test -d "$work/unpacked/${path##*/}"
	test ! -e "$root/$path"
	mv "$work/unpacked/${path##*/}" "$root/$path"
done

rm -rf "$work"
