package gpx

import (
	"fmt"

	"github.com/yuin/goldmark/parser"
	"gitlab.com/begraf/rueckblick/markdown/yamlblock"
)

var gpxMapCount = parser.NewContextKey()

func Count(pc parser.Context) int {
	if count, ok := pc.Get(gpxMapCount).(int); ok {
		return count
	}

	return 0
}

func ElementID(number int) string {
	return fmt.Sprintf("map-%02d", number)
}

type GPXSourceProvider interface {
	ProvideGPXSource(srcPath string) (string, bool)
}

type GPXAddin struct {
	sourceProvider GPXSourceProvider
}

func NewGPXAddin(sourceProvider GPXSourceProvider) *GPXAddin {
	return &GPXAddin{
		sourceProvider: sourceProvider,
	}
}

func (g *GPXAddin) AddinKey() string {
	return "gpx"
}

func (g *GPXAddin) Make(pc parser.Context) interface{} {
	path, ok := yamlblock.DocumentPath(pc)
	if !ok {
		panic("no document path")
	}

	return &gpxNode{
		documentPath: path,
		count:        getAndIncreaseGalleryCount(pc),
	}
}

func getAndIncreaseGalleryCount(pc parser.Context) int {
	count := Count(pc)

	pc.Set(gpxMapCount, count+1)

	return count
}
