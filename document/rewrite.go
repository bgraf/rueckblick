package document

import "github.com/bgraf/rueckblick/markdown/gpx"

type DocumentMediaRewriter interface {
	MakeRewriter(*Document) MediaRewriter
}

type MediaRewriter interface {
	gpx.GPXSourceProvider
}
