package data

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type periodDescriptor struct {
	From string
	To   string
}

type Period struct {
	Name string
	From time.Time
	To   time.Time
}

func LoadPeriods(path string) (periods []Period, err error) {
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

		periods = append(periods, Period{
			Name: k,
			From: from,
			To:   to,
		})
	}

	return
}
