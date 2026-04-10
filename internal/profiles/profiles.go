package profiles

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Profile represents a single git SSH identity profile.
type Profile struct {
	Name       string
	KeyFile    string
	OrgPattern string
}

// Load parses the INI-style profiles config file.
func Load(path string) ([]Profile, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	var result []Profile
	var current *Profile

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			if current != nil {
				result = append(result, *current)
			}
			current = &Profile{Name: line[1 : len(line)-1]}

		case strings.HasPrefix(line, "key_file=") && current != nil:
			current.KeyFile = strings.TrimSpace(strings.TrimPrefix(line, "key_file="))

		case strings.HasPrefix(line, "org_pattern=") && current != nil:
			current.OrgPattern = strings.TrimSpace(strings.TrimPrefix(line, "org_pattern="))
		}
	}

	if current != nil {
		result = append(result, *current)
	}

	return result, scanner.Err()
}

// Save serialises profiles back to the config file.
func Save(path string, profs []Profile) error {
	var sb strings.Builder
	for i, p := range profs {
		sb.WriteString(fmt.Sprintf("[%s]\n", p.Name))
		sb.WriteString(fmt.Sprintf("key_file=%s\n", p.KeyFile))
		sb.WriteString(fmt.Sprintf("org_pattern=%s\n", p.OrgPattern))
		if i < len(profs)-1 {
			sb.WriteString("\n")
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// ValidateSyntax returns whether all profiles have the required fields,
// along with a list of any issues found.
func ValidateSyntax(path string) (ok bool, issues []string) {
	profs, err := Load(path)
	if err != nil {
		return false, []string{fmt.Sprintf("parse error: %v", err)}
	}
	for _, p := range profs {
		if p.KeyFile == "" {
			issues = append(issues, fmt.Sprintf("[%s] is missing key_file", p.Name))
		}
		if p.OrgPattern == "" {
			issues = append(issues, fmt.Sprintf("[%s] is missing org_pattern", p.Name))
		}
	}
	return len(issues) == 0, issues
}
