// Package themes embeds the small offline catalog, never the optional artwork.
package themes

import _ "embed"

//go:embed catalog.json
var Catalog []byte
