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
	GitName    string
	GitEmail   string
	GHUser     string
	Sign       bool
	SignKey    string
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

		case strings.HasPrefix(line, "git_name=") && current != nil:
			current.GitName = strings.TrimSpace(strings.TrimPrefix(line, "git_name="))

		case strings.HasPrefix(line, "git_email=") && current != nil:
			current.GitEmail = strings.TrimSpace(strings.TrimPrefix(line, "git_email="))

		case strings.HasPrefix(line, "gh_user=") && current != nil:
			current.GHUser = strings.TrimSpace(strings.TrimPrefix(line, "gh_user="))

		case strings.HasPrefix(line, "sign=") && current != nil:
			current.Sign = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "sign=")), "true")

		case strings.HasPrefix(line, "sign_key=") && current != nil:
			current.SignKey = strings.TrimSpace(strings.TrimPrefix(line, "sign_key="))

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
		if p.GitName != "" {
			sb.WriteString(fmt.Sprintf("git_name=%s\n", p.GitName))
		}
		if p.GitEmail != "" {
			sb.WriteString(fmt.Sprintf("git_email=%s\n", p.GitEmail))
		}
		if p.GHUser != "" {
			sb.WriteString(fmt.Sprintf("gh_user=%s\n", p.GHUser))
		}
		if p.Sign {
			sb.WriteString("sign=true\n")
			if p.SignKey != "" {
				sb.WriteString(fmt.Sprintf("sign_key=%s\n", p.SignKey))
			}
		}
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
		if p.Sign && p.SignKey == "" {
			issues = append(issues, fmt.Sprintf("[%s] has signing enabled but no sign_key", p.Name))
		}
	}
	return len(issues) == 0, issues
}
