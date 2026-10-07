<?php
# Settings shared by every wiki in this image. MW_WIKI names the wiki, and its
# own settings are in settings/<MW_WIKI>.php. Secrets and anything else that
# differs per deployment come from the environment, set by the Quadlet's env
# file on the host; nothing secret is in this repository.

if ( !defined( 'MEDIAWIKI' ) ) {
	exit;
}

/** A required environment variable. */
function catramEnv( string $name ): string {
	$value = getenv( $name );
	if ( $value === false || $value === '' ) {
		throw new RuntimeException( "$name is not set" );
	}
	return $value;
}

$catramWiki = catramEnv( 'MW_WIKI' );
if ( !preg_match( '/^[a-z0-9]+$/', $catramWiki ) || !is_file( __DIR__ . "/settings/$catramWiki.php" ) ) {
	throw new RuntimeException( 'MW_WIKI does not name a wiki in settings/' );
}

## Server and paths
$wgServer = catramEnv( 'MW_SERVER' );
$wgScriptPath = '/w';
$wgArticlePath = '/wiki/$1';
$wgUsePathInfo = true;
$wgResourceBasePath = $wgScriptPath;

## Database: the account's own MariaDB, reached by container name.
$wgDBtype = 'mysql';
$wgDBserver = getenv( 'MW_DB_SERVER' ) ?: 'mariadb';
$wgDBname = catramEnv( 'MW_DB_NAME' );
$wgDBuser = catramEnv( 'MW_DB_USER' );
$wgDBpassword = catramEnv( 'MW_DB_PASSWORD' );

## Secrets
$wgSecretKey = catramEnv( 'MW_SECRET_KEY' );
$wgUpgradeKey = catramEnv( 'MW_UPGRADE_KEY' );

## Caches. The code is fixed inside the image, so APCu is enough.
$wgMainCacheType = CACHE_ACCEL;
$wgCacheDirectory = "$IP/cache";

## Jobs run in their own container (runJobs --wait), not during page views.
$wgJobRunRate = 0;

## Uploads and thumbnails
$wgUseImageMagick = true;
$wgImageMagickConvertCommand = '/usr/bin/convert';
$wgSVGConverter = 'rsvg';
# Missing thumbnails are made on request by thumb_handler.php.
$wgGenerateThumbnailOnParse = false;

## Skins and extensions every wiki uses. The rest are in settings/<wiki>.php.
wfLoadSkin( 'CologneBlue' );
wfLoadSkin( 'Modern' );
wfLoadSkin( 'MonoBook' );
wfLoadSkin( 'Vector' );

wfLoadExtension( 'Cite' );
wfLoadExtension( 'Interwiki' );
wfLoadExtension( 'OATHAuth' );
wfLoadExtension( 'ParserFunctions' );
wfLoadExtension( 'WikiEditor' );

require __DIR__ . "/settings/$catramWiki.php";
