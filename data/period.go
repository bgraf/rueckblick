package data

import (
	"os"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"gopkg.in/yaml.v2"
)

type periodDescriptor struct {
	From string
	To   string
}

func LoadPeriods(path string) (periods []document.Period, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	m := make(map[string]periodDescriptor)
	if err = yaml.Unmarshal(content, &m); err != nil {
		return
	}

	for k, v := range m {
		from, err := time.Parse("2006-01-02", v.From)
		if err != nil {
			return nil, err
		}

		to, err := time.Parse("2006-01-02", v.To)
		if err != nil {
			return nil, err
		}

		periods = append(periods, document.Period{
			Name: k,
			From: from,
			To:   to,
			Tag:  document.Tag{Raw: k, Category: "period"},
		})
	}

	return
}
