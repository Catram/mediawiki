# mediawiki

The container image for the Catram wikis, published as
`ghcr.io/catram/mediawiki`. It is deployed by
[ansible-playbooks](https://github.com/Catram/ansible-playbooks) as a rootless
Podman Quadlet, one account per wiki, behind nginx on the host.

## What is in the image

- MediaWiki core at a release branch (`MW_BRANCH`, now `REL1_46`), with the
  skins and extensions in `extensions.txt` at the same branch, and their
  Composer dependencies.
- The changes in `patches/`.
- PHP 8.3 with Apache, configured by `apache.conf` and `php.ini`. Only
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

## Building

Every push to `main` builds and pushes the image on the self-hosted runner,
and so does a weekly scheduled run, which picks up fixes backported to the
release branch. Each build is tagged `<branch>-<core commit>-<run number>`,
e.g. `REL1_46-1a2b3c4-17`; deployments pin one of these tags. The branch tag,
e.g. `REL1_46`, always points to the latest build.

To move to a new release, change `MW_BRANCH` in the workflow and the
`Containerfile`, and check that every patch still applies.
