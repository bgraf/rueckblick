package render

import (
	"github.com/bgraf/rueckblick/data/document"
	"github.com/lucasb-eyer/go-colorful"
)

type TagSet struct {
	colors map[string]colorful.Color
}

func NewTagSet() *TagSet {
	return &TagSet{
		colors: make(map[string]colorful.Color),
	}
}

func (ts *TagSet) HexColor(tag string) string {
	var (
		c  colorful.Color
		ok bool
	)

	normTag := document.NormalizeTagName(tag)
	if c, ok = ts.colors[normTag]; !ok {
		c = colorful.HappyColor()
		ts.colors[normTag] = c
	}

	return c.Hex()
}
