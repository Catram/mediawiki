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
- `LocalSettings.php` with the settings every wiki shares, and
  `settings/<wiki>.php` for each wiki.

## Running it

The image serves plain HTTP on port 80. It needs:

| Variable | Meaning |
|---|---|
| `MW_WIKI` | Which `settings/<wiki>.php` to load, e.g. `danmacu` |
| `MW_SERVER` | The wiki's URL, e.g. `https://danmacu.catram.org` |
| `MW_DB_SERVER` | Database host; defaults to `mariadb` |
| `MW_DB_NAME`, `MW_DB_USER`, `MW_DB_PASSWORD` | Database credentials |
| `MW_SECRET_KEY`, `MW_UPGRADE_KEY` | `$wgSecretKey`, `$wgUpgradeKey` |

and two writable mounts, owned by the container's `www-data` (UID 33):
`/var/www/html/w/images` for uploads and `/var/www/html/w/cache`.

Jobs are not run during page views. Run them in a second container from the
same image:

```sh
php /var/www/html/w/maintenance/run.php runJobs --wait
```

## Skins and extensions

Which ones a wiki loads is up to `LocalSettings.php` and
`settings/<wiki>.php`. Those the release bundles need nothing more; check the
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
