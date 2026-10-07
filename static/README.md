# Files at each wiki's site root

The files a wiki serves at the root of its site, such as `favicon.ico`, the
icons that `manifest.json` and `browserconfig.xml` list, `robots.txt`, or the
logo that `$wgLogo` names. Apache serves `/<file>` from
`static/<MW_WIKI>/<file>` if it exists, and otherwise from
`static/common/<file>`. Nothing in subfolders is served this way.

- `common/`: the files every wiki shares.
- `<wiki>/`: a wiki's own, which win over the shared ones of the same name.
