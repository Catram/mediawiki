<?php
# hucat.catram.org. Loaded by LocalSettings.php after the shared settings.
#
# TODO: port the rest of the current LocalSettings.php (site name, logo,
# language, permissions, uploads, namespaces, Math rendering, ...) without
# its secrets.

if ( !defined( 'MEDIAWIKI' ) ) {
	exit;
}

wfLoadExtension( 'CiteThisPage' );
wfLoadExtension( 'Disambiguator' );
wfLoadExtension( 'InputBox' );
wfLoadExtension( 'Math' );
