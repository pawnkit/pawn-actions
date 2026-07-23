package releaseset

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func ValidateGoMod(path string, content []byte) error {
	file, err := modfile.Parse(path, content, nil)
	if err != nil {
		return fmt.Errorf("release set: parse %s: %w", path, err)
	}
	if len(file.Replace) != 0 {
		return fmt.Errorf("release set: %s contains replace directives", path)
	}
	for _, requirement := range file.Require {
		if !strings.HasPrefix(requirement.Mod.Path, "github.com/pawnkit/") {
			continue
		}
		version := requirement.Mod.Version
		if !module.IsPseudoVersion(version) && versionPattern.MatchString(version) {
			continue
		}
		return fmt.Errorf("release set: %s uses unpublished PawnKit version %q", path, version)
	}
	if file.Module == nil {
		return errors.New("release set: go.mod has no module directive")
	}
	return nil
}
