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
  bundle, from [ExtensionDistributor](https://www.mediawiki.org/wiki/Special:ExtensionDistributor):
  each one's current tarball for the release's branch. Nothing runs Composer.
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

## Memory

These environment variables size PHP and Apache. The defaults suit a site
with little traffic; set any of them in the container's environment to change
it.

| Variable | Default | Sets |
|---|---|---|
| `PHP_MEMORY_LIMIT` | `256M` | `memory_limit`: the most one request may use |
| `PHP_OPCACHE_MEMORY` | `128` | `opcache.memory_consumption`, in MB: compiled code, shared by all processes |
| `PHP_APCU_SIZE` | `32M` | `apc.shm_size`: APCu's shared cache, used for `$wgMainCacheType = CACHE_ACCEL` |
| `APACHE_START_SERVERS` | `2` | `StartServers` |
| `APACHE_MIN_SPARE_SERVERS` | `1` | `MinSpareServers` |
| `APACHE_MAX_SPARE_SERVERS` | `3` | `MaxSpareServers`: idle processes beyond this are stopped |
| `APACHE_MAX_REQUEST_WORKERS` | `10` | `MaxRequestWorkers`: requests served at once; others wait |
| `APACHE_MAX_CONNECTIONS_PER_CHILD` | `500` | `MaxConnectionsPerChild`: a process is replaced after this many |

Each Apache process holds its own PHP memory, so the process counts matter
most. The shared caches only take memory as they fill.

## Skins and extensions

Which ones a wiki loads is up to `LocalSettings.php` and the wiki's own
settings. Those the release bundles need nothing more; check the
release's `extensions/` and `skins/` folders. Any other is a row in
`extensions.tsv`, tab-separated with a header: `path` (e.g.
`extensions/Disambiguator`), `url`, `sha256` and `note`. Listing one the
release already bundles fails the build.

The URL and SHA-256 pin an ExtensionDistributor tarball, which the build
downloads and checks. `update-extensions.sh` moves the pins: it asks
ExtensionDistributor's API, in one request, for each one's current tarball
for the release's branch (`REL1_46` for 1.46), the same that
[Special:ExtensionDistributor](https://www.mediawiki.org/wiki/Special:ExtensionDistributor)
hands out, and prints what moved. To add one, add a row with just its path
and run it:

```sh
sh update-extensions.sh
```

ExtensionDistributor keeps only the newest tarball of each branch, and its
API can name a tarball before it is built. Then the script changes nothing
and says so; run it again later.

## Building

Every push to `main` builds and pushes the image on a GitHub-hosted runner.
The base image is pinned by digest in the `Containerfile`, and Dependabot's
pull requests move it, so a build depends only on the commit. Each build
records its base image in the `org.opencontainers.image.base.digest` label.

A weekly scheduled run builds only if something changed since the published
image: it moves the extension pins with `update-extensions.sh` and commits
them if one moved, with what moved in the message, and compares the base
image in the `Containerfile` with the published image's label. A week with
nothing new builds nothing. If a tarball is not built yet, the run fails and
next week's tries again; run the workflow by hand to try sooner.

Each build is tagged `<version>-<run number>`, e.g. `1.46.2-4`; deployments
pin one of these tags. The series tag, e.g. `1.46`, always points to the
latest build.

To move to a new release, change `MW_VERSION` and `MW_SHA256` in the
`Containerfile`; the SHA-256 is in the release announcement, or:

```sh
curl -fsSL https://releases.wikimedia.org/mediawiki/1.46/mediawiki-1.46.2.tar.gz | sha256sum
```

For a new release series, also run `update-extensions.sh` to move every pin
to the new branch, and check that every patch still applies.
