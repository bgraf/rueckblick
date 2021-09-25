package gallery

import (
	"fmt"
	"time"

	"github.com/bgraf/rueckblick/markdown/yamlblock"
	"github.com/yuin/goldmark/parser"
)

var galleryCount = parser.NewContextKey()

func Count(pc parser.Context) int {
	if count, ok := pc.Get(galleryCount).(int); ok {
		return count
	}

	return 0
}

func ElementID(number int) string {
	return fmt.Sprintf("gallery-%02d", number)
}

type Options struct {
	ProvideSource func(galleryNo int, srcPath string, timestamp *time.Time) (string, bool)
}

type GalleryAddin struct {
	options *Options
}

func NewGalleryAddin(options *Options) yamlblock.Addin {
	return &GalleryAddin{
		options: options,
	}
}

func (g *GalleryAddin) AddinKey() string {
	return "gallery"
}

func (g *GalleryAddin) Make(pc parser.Context) interface{} {
	path, ok := yamlblock.DocumentPath(pc)
	if !ok {
		panic("no document path")
	}

	return &galleryNode{
		documentPath: path,
		count:        getAndIncreaseGalleryCount(pc),
		Path:         "photos",
		Include:      "*.jpg",
	}
}

func getAndIncreaseGalleryCount(pc parser.Context) int {
	count := Count(pc)

	pc.Set(galleryCount, count+1)

	return count
}
