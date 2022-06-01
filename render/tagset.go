package render

import (
	"github.com/bgraf/rueckblick/data/document"
	"github.com/lucasb-eyer/go-colorful"
)

type TagSet struct {
	colors map[string]colorful.Color
	tags   []document.Tag
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
		ts.tags = append(ts.tags, document.Tag{Raw: tag})
	}

	return c.Hex()
}

func (ts *TagSet) HexColors(tags ...string) []string {
	colors := make([]string, len(tags))

	for i, tag := range tags {
		colors[i] = ts.HexColor(tag)
	}

	return colors
}
