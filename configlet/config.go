package configlet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config is an Exercism track configuration.
type Config struct {
	path            string
	Active          bool `json:"active"`
	Deprecated      []string
	Exercises       []Exercise `json:"exercises"`
	Foregone        []string   `json:"foregone"`
	IgnorePattern   string     `json:"ignore_pattern"`
	Language        string     `json:"language"`
	Repository      string     `json:"repository"`
	Slug            string     `json:"slug"`
	SolutionPattern string     `json:"solution_pattern"`
}

// Exercise configures metadata about an implemented exercise.
// It's listed in the config in the order that the exercise will be
// delivered by the API.
type Exercise struct {
	Core       bool        `json:"core,omitempty"`
	Deprecated bool        `json:"deprecated,omitempty"`
	Difficulty int         `json:"difficulty,omitempty"`
	Topics     []string    `json:"topics,omitempty"`
	UnlockedBy interface{} `json:"unlocked_by,omitempty"`
	UUID       string      `json:"uuid"`
	Slug       string      `json:"slug"`
}

// Load loads an Exercism track configuration.
func Load(file string) (Config, error) {
	c := NewConfig()

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return c, fmt.Errorf("Unable to parse config: %s -- %s", file, err.Error())
	}

	return c, nil
}

// NewConfig creates a new Config with optional defaults set.
// Currently the only optional value is SolutionPattern which is used
// to work out if an exercise has a sample solution.
func NewConfig() Config {
	return Config{SolutionPattern: "[Ee]xample"}
}

// Slugs is the list of exercise identifiers for the track.
func (c Config) Slugs() []string {
	var slugs []string
	if len(c.Exercises) > 0 {
		for _, ex := range c.Exercises {
			slugs = append(slugs, ex.Slug)
		}
		return slugs
	}
	return slugs
}

func uniq(items []string) []string {
	uniques := map[string]bool{}
	for _, item := range items {
		uniques[item] = true
	}

	items = []string{}
	for unique := range uniques {
		items = append(items, unique)
	}
	return items
}
