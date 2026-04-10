package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"bipolar/internal/profiles"
	"bipolar/internal/shell"
)

// RunMainMenu is the top-level interactive loop.
func RunMainMenu() error {
	for {
		var choice string
		prompt := &survey.Select{
			Message: "Bipolar  —  Git SSH Profile Manager",
			Options: []string{
				"Configure Profiles",
				"Repair Bipolar",
				"Exit",
			},
		}

		if err := survey.AskOne(prompt, &choice); err != nil {
			printExitHint()
			return nil
		}

		switch choice {
		case "Configure Profiles":
			if err := runConfigureProfiles(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "Repair Bipolar":
			runRepair()
		case "Exit":
			printExitHint()
			return nil
		}
	}
}

func printExitHint() {
	_, rcFile, err := shell.DetectShell()
	if err != nil {
		return
	}
	fmt.Printf("\nPlease run \033[1msource %s\033[0m for changes to take effect.\n\n", rcFile)
}

// ── Configure Profiles ──────────────────────────────────────────────────────

func runConfigureProfiles() error {
	confPath := shell.ProfilesConfPath()

	for {
		profs, err := profiles.Load(confPath)
		if err != nil {
			return fmt.Errorf("cannot load profiles: %w", err)
		}

		// Build the menu: one entry per profile, then add, then back.
		options := []string{}
		for _, p := range profs {
			options = append(options, fmt.Sprintf("[%s]", p.Name))
		}
		options = append(options, "+ Add new profile", "Back")

		var choice string
		if err := survey.AskOne(&survey.Select{
			Message: "Configure Profiles",
			Options: options,
			Help:    "Select a profile to edit or delete it, or add a new one.",
		}, &choice); err != nil {
			return nil
		}

		switch {
		case choice == "Back":
			return nil

		case choice == "+ Add new profile":
			if err := addProfile(confPath, profs); err != nil && err != ErrCancelled {
				fmt.Printf("Error: %v\n", err)
			}

		default:
			name := strings.TrimPrefix(choice, "[")
			name = strings.TrimSuffix(name, "]")
			runProfileSubmenu(confPath, profs, name)
		}
	}
}

func runProfileSubmenu(confPath string, profs []profiles.Profile, name string) {
	var choice string
	if err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Profile  [%s]", name),
		Options: []string{
			fmt.Sprintf("Edit [%s]", name),
			fmt.Sprintf("Delete [%s]", name),
			"Back",
		},
	}, &choice); err != nil {
		return
	}

	switch {
	case strings.HasPrefix(choice, "Edit"):
		if err := editProfile(confPath, profs, name); err != nil && err != ErrCancelled {
			fmt.Printf("Error: %v\n", err)
		}
	case strings.HasPrefix(choice, "Delete"):
		if err := deleteProfile(confPath, profs, name); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}



func addProfile(confPath string, profs []profiles.Profile) error {
	fmt.Println()

	var name string
	if err := survey.AskOne(&survey.Input{
		Message: "Profile name:",
		Help:    "A short label, e.g. Work or PersonalGitHub",
	}, &name, survey.WithValidator(survey.Required)); err != nil {
		return ErrCancelled
	}
	name = strings.TrimSpace(name)

	for _, p := range profs {
		if strings.EqualFold(p.Name, name) {
			fmt.Printf("\033[33mProfile [%s] already exists — use Edit to modify it.\033[0m\n\n", name)
			return nil
		}
	}

	var orgPattern string
	if err := survey.AskOne(&survey.Input{
		Message: "Org / username pattern:",
		Help:    "Exact string or regex. Examples: myorg   ^(alice|bob)$   .*",
	}, &orgPattern, survey.WithValidator(survey.Required)); err != nil {
		return ErrCancelled
	}

	fmt.Println("\nSelect SSH private key file:")
	keyFile, err := PickFile(sshDir())
	if err != nil {
		return ErrCancelled
	}
	keyFile = toTildePath(keyFile)

	newProf := profiles.Profile{
		Name:       name,
		KeyFile:    keyFile,
		OrgPattern: strings.TrimSpace(orgPattern),
	}

	profs = append(profs, newProf)

	if err := profiles.Save(confPath, profs); err != nil {
		return fmt.Errorf("cannot save profiles: %w", err)
	}

	fmt.Printf("\n\033[32m✓ Profile [%s] added.\033[0m\n\n", name)
	return nil
}

func editProfile(confPath string, profs []profiles.Profile, name string) error {
	idx := -1
	for i, p := range profs {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("profile [%s] not found", name)
	}

	p := profs[idx]
	fmt.Println()

	var newName string
	if err := survey.AskOne(&survey.Input{
		Message: "Profile name:",
		Default: p.Name,
	}, &newName, survey.WithValidator(survey.Required)); err != nil {
		return ErrCancelled
	}

	var orgPattern string
	if err := survey.AskOne(&survey.Input{
		Message: "Org / username pattern:",
		Default: p.OrgPattern,
		Help:    "Exact string or regex. Examples: myorg   ^(alice|bob)$   .*",
	}, &orgPattern, survey.WithValidator(survey.Required)); err != nil {
		return ErrCancelled
	}

	var changeKey bool
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Change key file? (current: %s)", p.KeyFile),
		Default: false,
	}, &changeKey); err != nil {
		return ErrCancelled
	}

	keyFile := p.KeyFile
	if changeKey {
		fmt.Println("\nSelect SSH private key file:")
		picked, err := PickFile(sshDir())
		if err != nil {
			return ErrCancelled
		}
		keyFile = toTildePath(picked)
	}

	profs[idx] = profiles.Profile{
		Name:       strings.TrimSpace(newName),
		KeyFile:    keyFile,
		OrgPattern: strings.TrimSpace(orgPattern),
	}

	if err := profiles.Save(confPath, profs); err != nil {
		return fmt.Errorf("cannot save profiles: %w", err)
	}

	fmt.Printf("\n\033[32m✓ Profile [%s] updated.\033[0m\n\n", newName)
	return nil
}

func deleteProfile(confPath string, profs []profiles.Profile, name string) error {
	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Delete profile [%s]? This cannot be undone.", name),
		Default: false,
	}, &confirm); err != nil || !confirm {
		return nil
	}

	var kept []profiles.Profile
	for _, p := range profs {
		if p.Name != name {
			kept = append(kept, p)
		}
	}

	if err := profiles.Save(confPath, kept); err != nil {
		return fmt.Errorf("cannot save profiles: %w", err)
	}

	fmt.Printf("\033[32m✓ Profile [%s] deleted.\033[0m\n\n", name)
	return nil
}

// ── Repair Bipolar ───────────────────────────────────────────────────────────

func runRepair() {
	fmt.Printf("\n\033[1mRunning Bipolar diagnostics...\033[0m\n\n")

	shellName, rcFile, err := shell.DetectShell()
	if err != nil {
		fmt.Printf("  \033[31m✗ Shell detection failed: %v\033[0m\n\n", err)
		return
	}
	fmt.Printf("  Shell: \033[1m%s\033[0m  |  RC file: \033[1m%s\033[0m\n\n", shellName, rcFile)

	// Check the rc function block.
	rcExists, rcUpToDate, err := shell.CheckRCFile(rcFile)
	if err != nil {
		fmt.Printf("  \033[31m✗ RC file check error: %v\033[0m\n", err)
	} else if !rcExists {
		fmt.Printf("  \033[31m✗ git-dispatch function not found in %s\033[0m\n", rcFile)
		var fix bool
		_ = survey.AskOne(&survey.Confirm{Message: "Install it now?", Default: true}, &fix)
		if fix {
			if err := shell.InstallToRCFile(rcFile); err != nil {
				fmt.Printf("    \033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Printf("    \033[32m✓ Installed.  Run: source %s\033[0m\n", rcFile)
			}
		}
	} else if !rcUpToDate {
		fmt.Printf("  \033[33m⚠ git-dispatch function in %s is outdated\033[0m\n", rcFile)
		var fix bool
		_ = survey.AskOne(&survey.Confirm{Message: "Update it to the latest version?", Default: true}, &fix)
		if fix {
			if err := shell.UpdateInRCFile(rcFile); err != nil {
				fmt.Printf("    \033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Printf("    \033[32m✓ Updated.  Run: source %s\033[0m\n", rcFile)
			}
		}
	} else {
		fmt.Printf("  \033[32m✓ git-dispatch function is present and up to date\033[0m\n")
	}

	// Check profiles config.
	confPath := shell.ProfilesConfPath()
	if !shell.CheckProfilesConf() {
		fmt.Printf("  \033[31m✗ Profiles config not found at %s\033[0m\n", confPath)
		var fix bool
		_ = survey.AskOne(&survey.Confirm{Message: "Create a default config?", Default: true}, &fix)
		if fix {
			if err := shell.CreateDefaultProfilesConf(); err != nil {
				fmt.Printf("    \033[31mError: %v\033[0m\n", err)
			} else {
				fmt.Printf("    \033[32m✓ Created default profiles config\033[0m\n")
			}
		}
	} else {
		valid, issues := profiles.ValidateSyntax(confPath)
		if !valid {
			fmt.Printf("  \033[33m⚠ Profiles config has issues:\033[0m\n")
			for _, issue := range issues {
				fmt.Printf("      - %s\n", issue)
			}
		} else {
			profs, _ := profiles.Load(confPath)
			fmt.Printf("  \033[32m✓ Profiles config is valid  (%d profile(s))\033[0m\n", len(profs))
		}
	}

	fmt.Println()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func sshDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh")
}

// toTildePath replaces the home directory prefix with "~".
func toTildePath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}
