# mediawiki

The container image for the Catram wikis, published as
`ghcr.io/catram/mediawiki`. It is deployed by
[ansible-playbooks](https://github.com/Catram/ansible-playbooks) as a rootless
Podman Quadlet, one account per wiki, behind nginx on the host.

## What is in the image

- The MediaWiki release in `MW_VERSION` (now 1.46.2), from
  releases.wikimedia.org and checked against `MW_SHA256`, both in the
  `Containerfile`. The release brings `vendor/` and its bundled skins and
  extensions; the web installer, `mw-config/`, is left out.
- The skins and extensions in `extensions.tsv` that the release does not
  bundle, from [ExtensionDistributor](https://www.mediawiki.org/wiki/Special:ExtensionDistributor),
  each checked against its SHA-256. Nothing runs Composer.
- The changes in `patches/`.
- PHP 8.4 with Apache, configured by `apache.conf` and `php.ini`. Only
  MediaWiki's entry points run as PHP.
- No `LocalSettings.php`: each wiki's deployment mounts its own, with the
  skins and extensions it loads and its secrets.
- `static/`: the files at the wikis' site roots, such as logos and the
  favicon (see `static/README.md`).

## Running it

The image serves plain HTTP on port 80. It needs `MW_WIKI`, the wiki's name
(e.g. `danmacu`), which picks its files in `static/`, and these mounts:

| Path | Contents |
|---|---|
| `/var/www/html/w/LocalSettings.php` | The wiki's settings. Read-only; `www-data` must be able to read it |
| `/var/www/html/w/images` | Uploads, writable by the container's `www-data` (UID 33) |
| `/var/www/html/w/cache` | Cache, writable by `www-data`; a tmpfs is best, so each start begins empty |

The Quadlets in ansible-playbooks render each wiki's `LocalSettings.php` from
a template and map `www-data` to the wiki's account on the host
(`UserNS=keep-id:uid=33,gid=33`), so it reads the installed file through the
account's group and owns the uploads as the account.

The settings are expected to set `$wgCacheDirectory = "$IP/cache"`, and
`$wgJobRunRate = 0` with the jobs run in a second container from the same
image:

```sh
php /var/www/html/w/maintenance/run.php runJobs --wait
```

## Skins and extensions

Which ones a wiki loads is up to `LocalSettings.php` and the wiki's own
settings. Those the release bundles need nothing more; check the
release's `extensions/` and `skins/` folders. Any other is a row in
`extensions.tsv`, tab-separated with a header: `path` (e.g.
`extensions/Disambiguator`), `url`, `sha256` and `note`. The tarball must hold
one folder named like the last part of the path. Listing one the release
already bundles fails the build.

Take the URL for the release's branch (`REL1_46` for 1.46) from
ExtensionDistributor, or for many at once from its API:

```sh
curl -s 'https://www.mediawiki.org/w/api.php?action=query&list=extdistbranches&format=json&edbexts=Disambiguator|ProofreadPage&edbskins=Modern'
```

and its SHA-256:

```sh
curl -fsSL <url> | sha256sum
```

ExtensionDistributor keeps only the newest tarball of each branch. When a fix
is backported, the pinned URL disappears and the build fails with a 404 until
the row is updated.

## Building

Every push to `main` builds and pushes the image on a GitHub-hosted runner,
and so does a weekly scheduled run, which picks up fixes to the PHP image.
Each build is tagged `<version>-<run number>`, e.g. `1.46.2-4`; deployments
pin one of these tags. The series tag, e.g. `1.46`, always points to the
latest build.

To move to a new release, change `MW_VERSION` and `MW_SHA256` in the
`Containerfile`; the SHA-256 is in the release announcement, or:

```sh
curl -fsSL https://releases.wikimedia.org/mediawiki/1.46/mediawiki-1.46.2.tar.gz | sha256sum
```

For a new release series, also update every URL in `extensions.tsv` to the
new branch, and check that every patch still applies.
