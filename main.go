package main

import (
	"fmt"
	"os"

	"github.com/exercism/configlet/configlet"
)

// Check identifies configuration problems.
type Check func() ([]string, error)

// ConfigError defines the error message for a Check.
type ConfigError struct {
	check Check
	msg   string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: configlet path/to/problems/repository")
		os.Exit(1)
	}

	path := os.Args[1]
	fmt.Printf("Evaluating %s\n", path)

	track := configlet.NewTrack(path)

	hasErrors := false
	if !track.HasValidConfig() {
		hasErrors = true
		fmt.Println("-> config.json is invalid")
	}

	configErrors := []ConfigError{
		ConfigError{
			check: track.MissingProblems,
			msg:   "-> No directory found for %v.\n",
		},
		ConfigError{
			check: track.UnconfiguredProblems,
			msg:   "-> config.json does not include %v.\n",
		},
		ConfigError{
			check: track.ProblemsLackingExample,
			msg:   "-> missing example solution in %v.\n",
		},
		ConfigError{
			check: track.ForegoneViolations,
			msg:   "-> %v should not be implemented.\n",
		},
		ConfigError{
			check: track.DuplicateSlugs,
			msg:   "-> %v found in multiple categories.\n",
		},
	}

	for _, configError := range configErrors {
		result, err := configError.check()

		if err != nil {
			hasErrors = true
			fmt.Errorf("-> %v", err)
		}

		if len(result) > 0 {
			hasErrors = true
			fmt.Printf(configError.msg, result)
		}
	}

	if hasErrors {
		os.Exit(1)
	}

	fmt.Println("... OK")
}
