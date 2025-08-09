package data

import "strings"

type Tag struct {
	Raw      string
	Category string
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
