package document

import "gitlab.com/begraf/rueckblick/markdown/gpx"

type DocumentMediaRewriter interface {
	MakeRewriter(*Document) MediaRewriter
}

type MediaRewriter interface {
	gpx.GPXSourceProvider
}
