package serve

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"gitlab.com/begraf/rueckblick/document"
	"gitlab.com/begraf/rueckblick/markdown/gpx"
)

type rewriter struct {
	bySrc map[string]string
	byDst map[string]string
}

func (r *rewriter) IDFromPath(srcPath string) string {
	var err error
	srcPath, err = filepath.Abs(srcPath)
	if err != nil {
		panic(err)
	}

	if dstName, ok := r.bySrc[srcPath]; ok {
		return dstName
	}

	guid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	dstName := guid.String()
	r.bySrc[srcPath] = dstName
	r.byDst[dstName] = srcPath

	return dstName
}

func (r *rewriter) PathFromID(id string) (string, bool) {
	if src, ok := r.byDst[id]; ok {
		return src, true
	}

	return "", false
}

func newRewriter() *rewriter {
	return &rewriter{
		bySrc: make(map[string]string),
		byDst: make(map[string]string),
	}
}

type documentMediaRewriter struct {
	rewriter *rewriter
}

func newDocumentRewriter(m *rewriter) *documentMediaRewriter {
	return &documentMediaRewriter{m}
}

func (m *documentMediaRewriter) MakeRewriter(doc *document.Document) document.MediaRewriter {
	return &mediaRewriter{m.rewriter, doc}
}

type mediaRewriter struct {
	rewriter *rewriter
	document *document.Document
}

func (m *mediaRewriter) ProvideGPXSource(srcPath string) (string, bool) {
	srcPath = pathAbsOrPanic(srcPath)
	resourceID := m.rewriter.IDFromPath(srcPath)

	mapNumber := len(m.document.Maps)

	m.document.Maps = append(
		m.document.Maps,
		document.GXPMap{
			GPXPath:    srcPath,
			ResourceID: resourceID,
			ElementID:  gpx.ElementID(mapNumber),
		},
	)

	return m.gpxRoute(resourceID), true
}

func (m *mediaRewriter) gpxRoute(dstName string) string {
	return fmt.Sprintf("/gpx/%s", dstName)
}

func pathAbsOrPanic(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	return path
}
