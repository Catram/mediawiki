<?php
# danmacu.catram.org. Loaded by LocalSettings.php after the shared settings.
#
# TODO: port the rest of the current LocalSettings.php (site name, logo,
# language, permissions, uploads, namespaces, ...) without its secrets.

if ( !defined( 'MEDIAWIKI' ) ) {
	exit;
}

wfLoadExtension( 'CodeEditor' );
wfLoadExtension( 'Gadgets' );
wfLoadExtension( 'LabeledSectionTransclusion' );
wfLoadExtension( 'PagedTiffHandler' );
wfLoadExtension( 'PdfHandler' );
wfLoadExtension( 'Poem' );
wfLoadExtension( 'ProofreadPage' );
wfLoadExtension( 'SyntaxHighlight_GeSHi' );
