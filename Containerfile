# MediaWiki for the Catram wikis: a release with its bundled skins and
# extensions, the others in extensions.tsv, and patches/ applied, on PHP and
# Apache. One image serves every wiki; MW_WIKI picks the settings at run time.

FROM docker.io/library/php:8.4-apache

# Tools that MediaWiki and the extensions call: ImageMagick for thumbnails,
# Ghostscript and poppler for PdfHandler, libtiff for PagedTiffHandler, rsvg
# for SVG, Python for SyntaxHighlight's bundled Pygments.
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends \
		ghostscript \
		imagemagick \
		librsvg2-bin \
		libtiff-tools \
		poppler-utils \
		python3 \
	; \
	rm -rf /var/lib/apt/lists/*

# PHP extensions. The build dependencies are removed again afterwards; only
# the libraries the compiled extensions link to are kept.
RUN set -eux; \
	savedAptMark="$(apt-mark showmanual)"; \
	apt-get update; \
	apt-get install -y --no-install-recommends libicu-dev; \
	docker-php-ext-install -j "$(nproc)" calendar intl mysqli opcache; \
	pecl install apcu; \
	docker-php-ext-enable apcu; \
	rm -rf /tmp/pear; \
	apt-mark auto '.*' > /dev/null; \
	apt-mark manual $savedAptMark; \
	ldd "$(php -r 'echo ini_get("extension_dir");')"/*.so \
		| awk '/=>/ { so = $(NF-1); if (index(so, "/usr/local/") == 1) { next }; gsub("^/(usr/)?", "", so); printf "*%s\n", so }' \
		| sort -u \
		| xargs -r dpkg-query --search \
		| cut -d: -f1 \
		| sort -u \
		| xargs -rt apt-mark manual; \
	apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false; \
	rm -rf /var/lib/apt/lists/*

RUN a2enmod remoteip rewrite
COPY apache.conf /etc/apache2/sites-available/000-default.conf
COPY php.ini /usr/local/etc/php/conf.d/mediawiki.ini

# The release tarball from releases.wikimedia.org, with vendor/ and the
# bundled skins and extensions, and the ones pinned in extensions.tsv, each
# download checked against its SHA-256 (see install.sh). Then the local
# changes, as patches relative to w/, applied in name order.
ARG MW_VERSION=1.46.2
ARG MW_SHA256=8f7f937f8bbc1acd4cef1c5362a95f248dcd242020f2588798eb2f73ac2e69ba
WORKDIR /var/www/html
COPY install.sh extensions.tsv /tmp/build/
COPY patches/ /tmp/build/patches/
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends wget patch; \
	sh /tmp/build/install.sh "$MW_VERSION" "$MW_SHA256" /var/www/html/w; \
	for p in /tmp/build/patches/*.patch; do \
		[ -e "$p" ] || continue; \
		patch -d w -p1 --forward --batch < "$p"; \
	done; \
	rm -rf /tmp/build; \
	apt-get purge -y --auto-remove wget patch; \
	rm -rf /var/lib/apt/lists/*

COPY LocalSettings.php w/LocalSettings.php
COPY settings/ w/settings/

# Code stays owned by root and read-only to Apache. These are mounted from the
# host, per wiki, and must be writable by www-data.
RUN set -eux; \
	mkdir -p w/images w/cache; \
	chown www-data:www-data w/images w/cache
VOLUME ["/var/www/html/w/images", "/var/www/html/w/cache"]
