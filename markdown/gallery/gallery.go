package gallery

import (
	"fmt"

	"github.com/yuin/goldmark/parser"
	"github.com/bgraf/rueckblick/markdown/yamlblock"
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

type GalleryAddin struct {
}

func NewGalleryAddin() yamlblock.Addin {
	return &GalleryAddin{}
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
	}
}

func getAndIncreaseGalleryCount(pc parser.Context) int {
	count := Count(pc)

	pc.Set(galleryCount, count+1)

	return count
}
