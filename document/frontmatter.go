package document

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v2"
	"time"
)

type FrontMatter struct {
	Title    string              `yaml:"title"`
	Date     YamlDate            `yaml:"date"`
	Author   string              `yaml:"author"`
	Preview  string              `yaml:"preview,omitempty"`
	Abstract string              `yaml:"abstract,omitempty"`
	GUID     uuid.UUID           `yaml:"guid,omitempty"`
	Tags     map[string][]string `yaml:"tags,omitempty"`
}

func readFrontMatter(doc *Document, source []byte) ([]byte, error) {
	fmSource, mdSource, err := findFrontMatterSource(source)
	if err != nil {
		return source, fmt.Errorf("read front matter: %w", err)
	}

	fm := FrontMatter{}

	err = yaml.Unmarshal(fmSource, &fm)
	if err != nil {
		return source, fmt.Errorf("parse YAML: %w", err)
	}

	if fm.GUID.ID() == 0 {
		fm.GUID, err = uuid.NewRandom()
		if err != nil {
			return source, fmt.Errorf("new random UUID")
		}
	}

	doc.Title = fm.Title
	doc.Date = time.Time(fm.Date)
	doc.Abstract = fm.Abstract
	doc.Preview = fm.Preview
	doc.GUID = fm.GUID

	for category, names := range fm.Tags {
		for _, name := range names {
			doc.Tags = append(
				doc.Tags,
				Tag{
					Raw:      name,
					Category: category,
				},
			)
		}
	}

	doc.HasFrontMatter = true

	return mdSource, nil
}

type YamlDate time.Time

func (t *YamlDate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var txt string
	err := unmarshal(&txt)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", txt)
	if err != nil {
		return err
	}

	*t = YamlDate(date)
	return nil
}

func (t YamlDate) MarshalYAML() (interface{}, error) {
	ret := time.Time(t).Format("2006-01-02")
	return ret, nil
}

func findFrontMatterSource(source []byte) ([]byte, []byte, error) {
	nSkipWhite := util.FirstNonSpacePosition(source)
	if !startsWithFrontMatterMarker(source[nSkipWhite:]) {
		// Assume no front-matter
		return nil, source, nil
	}

	startPos := nSkipWhite + 3
	endPos, ok := findEndPos(source[startPos:])
	if !ok {
		return nil, source, fmt.Errorf("no front matter ending indicator")
	}
	endPos += nSkipWhite + 3

	return source[startPos:endPos], source[endPos+3:], nil
}

func findEndPos(source []byte) (int, bool) {
	ub := len(source) - 3
	for i := 0; i < ub; i++ {
		if startsWithFrontMatterMarker(source[i:]) {
			return i, true
		}
	}

	return 0, false
}

func startsWithFrontMatterMarker(source []byte) bool {
	return bytes.HasPrefix(source, []byte{'-', '-', '-'})
}
