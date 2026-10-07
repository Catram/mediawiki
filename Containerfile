# MediaWiki for the Catram wikis: core and the skins and extensions in
# extensions.txt at one release branch, with patches/ applied, on PHP and
# Apache. One image serves every wiki; MW_WIKI picks the settings at run time.

FROM docker.io/library/php:8.3-apache

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

COPY --from=docker.io/library/composer:2 /usr/bin/composer /usr/local/bin/composer

# Core at the release branch, or at MW_COMMIT on it when the build pins one.
ARG MW_BRANCH=REL1_46
ARG MW_COMMIT=
ARG GERRIT=https://gerrit.wikimedia.org/r/mediawiki
WORKDIR /var/www/html
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends git patch unzip; \
	git clone --depth 1 --branch "$MW_BRANCH" "$GERRIT/core.git" w; \
	if [ -n "$MW_COMMIT" ]; then \
		git -C w fetch --depth 1 origin "$MW_COMMIT"; \
		git -C w checkout --detach "$MW_COMMIT"; \
	fi; \
	git -C w rev-parse HEAD > w/.core-commit

COPY extensions.txt composer.local.json /tmp/build/
RUN set -eux; \
	grep -Ev '^[[:space:]]*(#|$)' /tmp/build/extensions.txt | while read -r path; do \
		git clone --depth 1 --branch "$MW_BRANCH" "$GERRIT/$path.git" "w/$path"; \
	done; \
	cp /tmp/build/composer.local.json w/; \
	COMPOSER_ALLOW_SUPERUSER=1 composer --working-dir=w update --no-dev --no-interaction --optimize-autoloader

# Local changes, as patches relative to w/, applied in name order.
COPY patches/ /tmp/build/patches/
RUN set -eux; \
	for p in /tmp/build/patches/*.patch; do \
		[ -e "$p" ] || continue; \
		patch -d w -p1 --forward --batch < "$p"; \
	done

# The build tools and history are not needed at run time.
RUN set -eux; \
	find w -name .git -prune -exec rm -rf {} +; \
	rm -rf /tmp/build /root/.composer /usr/local/bin/composer; \
	apt-get purge -y --auto-remove git patch unzip; \
	rm -rf /var/lib/apt/lists/*

COPY LocalSettings.php w/LocalSettings.php
COPY settings/ w/settings/

# Code stays owned by root and read-only to Apache. These are mounted from the
# host, per wiki, and must be writable by www-data.
RUN set -eux; \
	mkdir -p w/images w/cache; \
	chown www-data:www-data w/images w/cache
VOLUME ["/var/www/html/w/images", "/var/www/html/w/cache"]
