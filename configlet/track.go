package configlet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Track is a collection of Exercism exercises in a given programming language.
type Track struct {
	path string
	dirs map[string]string
}

// NewTrack is an exercism language track that lives at path.
// It uses the config.json in the root of the track to figure
// out which exercises a track contains.
func NewTrack(path string) (Track, error) {
	t := Track{path: path, dirs: map[string]string{}}

	slugs, err := t.Slugs()
	if err != nil {
		return t, err
	}

	for slug := range slugs {
		path := filepath.Join(t.path, "exercises", slug)

		fi, err := os.Stat(path)
		if err == nil && fi.IsDir() && isHiddenDir(fi.Name()) {
			t.dirs[slug] = path
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			return t, err
		}
	}

	return t, nil
}

// Config loads a track's configuration.
func (t Track) Config() (Config, error) {
	c, err := Load(t.configFile())
	if err != nil {
		return c, err
	}
	return c, nil
}

// HasValidConfig lints the JSON file.
func (t Track) HasValidConfig() bool {

	c, err := t.Config()
	// re-marshall json with 2 space indent
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("b = %+v\n", string(b))
	return err == nil
}

// Problems lists all the problem specifications that a track has implemented exercises for.
func (t Track) Problems() (map[string]struct{}, error) {
	problems := make(map[string]struct{})

	c, err := t.Config()
	if err != nil {
		return problems, err
	}

	for _, problem := range c.Slugs() {
		problems[problem] = struct{}{}
	}

	return problems, nil
}

// Slugs is a list of all problems mentioned in the config.
func (t Track) Slugs() (map[string]struct{}, error) {
	slugs := make(map[string]struct{})

	c, err := t.Config()
	if err != nil {
		return slugs, err
	}

	for _, slug := range c.Slugs() {
		slugs[slug] = struct{}{}
	}

	for _, slug := range c.Deprecated {
		slugs[slug] = struct{}{}
	}

	for _, slug := range c.Foregone {
		slugs[slug] = struct{}{}
	}
	return slugs, nil
}

// Dirs is a list of all the relevant directories.
func (t Track) Dirs() (map[string]struct{}, error) {
	dirs := make(map[string]struct{})

	path := filepath.Join(t.path, "exercises")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return dirs, nil
		}
		return dirs, err
	}

	infos, err := ioutil.ReadDir(filepath.Join(t.path, "exercises"))
	if err != nil {
		return dirs, err
	}

	for _, info := range infos {
		if info.IsDir() && info.Name() != "exercises" && isHiddenDir(info.Name()) {
			dirs[info.Name()] = struct{}{}
		}
	}

	return dirs, nil
}

// MissingProblems identify problems lacking an implementation.
// This will complain if the problem slug is listed in the configuration,
// but there is no corresponding directory for it.
func (t Track) MissingProblems() ([]string, error) {
	dirs, err := t.Dirs()
	if err != nil {
		return []string{}, err
	}

	problems, err := t.Problems()
	if err != nil {
		return []string{}, err
	}

	omissions := make([]string, 0, len(problems))

	for problem := range problems {
		_, present := dirs[problem]
		if !present {
			omissions = append(omissions, problem)
		}
	}
	return omissions, nil
}

// UnconfiguredProblems identifies unlisted implementations.
// This will complain if a directory exists, but is not mentioned
// anywhere in the config file.
func (t Track) UnconfiguredProblems() ([]string, error) {
	dirs, err := t.Dirs()
	if err != nil {
		return []string{}, err
	}

	slugs, err := t.Slugs()
	if err != nil {
		return []string{}, err
	}

	omissions := make([]string, 0, len(slugs))

	for dir := range dirs {
		_, present := slugs[dir]
		if !present {
			omissions = append(omissions, dir)
		}
	}
	return omissions, nil
}

// ProblemsLackingExample identifies implementations without a solution.
// This will often be triggered because the implementation's sample solution
// is not named something with example. This is particularly critical since
// any file that is in a path not named /[Ee]xample/ will be served by the API,
// showing the user a possible solution before they have solved the problem
// themselves.
func (t Track) ProblemsLackingExample() ([]string, error) {
	c, err := t.Config()
	if err != nil {
		return nil, err
	}

	var issues []string

	for _, problem := range c.Slugs() {
		path := t.dirs[problem]
		if path == "" {
			continue
		}

		files, err := findAllFiles(path)
		if err != nil {
			return issues, err
		}
		found, err := t.hasExampleFile(files)
		if !found {
			issues = append(issues, problem)
		}
	}

	return issues, nil
}

// ForegoneViolations indentifies implementations that should not be included.
// This could be because the problem is too trivial, ridiculously non-trivial,
// or simply uninteresting.
func (t Track) ForegoneViolations() ([]string, error) {
	problems := []string{}

	c, err := t.Config()
	if err != nil {
		return problems, err
	}

	dirs, err := t.Dirs()
	if err != nil {
		return problems, err
	}

	violations := make([]string, 0, len(dirs))

	for _, problem := range c.Foregone {
		_, present := dirs[problem]
		if present {
			violations = append(violations, problem)
		}
	}
	return violations, nil
}

// DuplicateSlugs detects slugs in multiple config categories.
// If a problem is deprecated, it means that we have the files for it,
// we're just not serving it in the default response.
// If a slug is foregone, it means that we've chosen not to implement it,
// and it should not have a directory.
func (t Track) DuplicateSlugs() ([]string, error) {
	counts := make(map[string]int)

	c, err := t.Config()
	if err != nil {
		return []string{}, err
	}

	for _, slug := range c.Slugs() {
		counts[slug] = counts[slug] + 1
	}

	for _, slug := range c.Deprecated {
		counts[slug] = counts[slug] + 1
	}

	for _, slug := range c.Foregone {
		counts[slug] = counts[slug] + 1
	}

	dupes := make([]string, 0, len(counts))
	for slug, count := range counts {
		if count > 1 {
			dupes = append(dupes, slug)
		}
	}
	sort.Strings(dupes)

	return dupes, nil
}

func (t Track) configFile() string {
	return fmt.Sprintf("%s/config.json", t.path)
}

func (t Track) hasExampleFile(files []string) (bool, error) {
	c, err := t.Config()
	if err != nil {
		return false, err
	}

	r, err := regexp.Compile(c.SolutionPattern)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		matches := r.Find([]byte(file))
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func findAllFiles(path string) ([]string, error) {
	files := []string{}

	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return files, err
	}

	for _, info := range infos {
		subPath := fmt.Sprintf("%s/%s", path, info.Name())
		if info.IsDir() {
			subFiles, err := findAllFiles(subPath)
			if err != nil {
				return files, err
			}
			files = append(files, subFiles...)
		} else {
			files = append(files, subPath)
		}
	}
	return files, nil
}

func isHiddenDir(name string) bool {
	return name[0] != 46
}
