package document

import "strings"

type Tag struct {
	Raw      string
	Category string
}

func (t Tag) HasCategory() bool {
	return len(t.Category) > 0
}

func (t Tag) String() string {
	return t.Raw
}

func (t Tag) Normalize() string {
	return NormalizeTagName(t.Raw)
}

func NormalizeTagName(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	return tag
}
