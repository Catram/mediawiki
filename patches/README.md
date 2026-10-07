# Patches

Local changes to MediaWiki core, skins or extensions. Each `*.patch` file is
applied in name order to the checkout in `w/` with `patch -p1`, so paths are
relative to the MediaWiki root, for example `a/extensions/Cite/...`.

Name them with a number first (`0001-short-description.patch`) to fix the
order. A patch that no longer applies to a new release branch fails the
build, so it is noticed instead of silently dropped.
