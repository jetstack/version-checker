package checker

import (
	"fmt"
	"regexp"
	"strings"
)

type Substitution struct {
	Pattern    *regexp.Regexp
	Substitute string
	All        bool
}

func NewSubstitutionFromSedCommand(sedCommand string) (*Substitution, error) {
	pattern, substitute, flags, err := splitSedSubstitutionCommand(sedCommand)
	if err != nil {
		return nil, err
	}
	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("sed command for substitution has regex that does not compile: %s %w", pattern, err)
	}
	var all bool
	if flags == "g" {
		all = true
	} else if flags != "" {
		return nil, fmt.Errorf("sed command for substitution only supports the 'g' flag: %s", flags)
	}

	return &Substitution{
		Pattern:    compiledPattern,
		Substitute: substitute,
		All:        all,
	}, nil
}

func splitSedSubstitutionCommand(sedCommand string) (string, string, string, error) {
	if len(sedCommand) < 4 {
		return "", "", "", fmt.Errorf("sed command for substitution seems to short: %s", sedCommand)
	}
	if sedCommand[0] != 's' {
		return "", "", "", fmt.Errorf("sed command for substitution should start with s: %s", sedCommand)
	}
	separator := regexp.QuoteMeta(sedCommand[1:2])
	group := fmt.Sprintf(`((?:.|\\[%s])*)`, separator)
	pattern := `^s` + separator + group + separator + group + separator + group + `$`
	matcher, err := regexp.Compile(pattern)
	if err != nil {
		return "", "", "", fmt.Errorf("regexp to parse sed command for substitution does not compile: %s %w", pattern, err)
	}
	submatches := matcher.FindStringSubmatch(sedCommand)
	if len(submatches) != 4 {
		return "", "", "", fmt.Errorf("sed command for substitution could not be parsed: %s", pattern)
	}
	return strings.ReplaceAll(submatches[1], "\\"+separator, separator),
		strings.ReplaceAll(submatches[2], "\\"+separator, separator),
		submatches[3], nil
}
