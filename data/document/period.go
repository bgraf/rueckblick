package document

import (
	"time"
)

type Period struct {
	Name string
	Tag  Tag
	From time.Time
	To   time.Time
}
