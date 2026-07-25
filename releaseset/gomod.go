package releaseset

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

type GoModule struct {
	Path         string
	Repository   string
	Dependencies []GoDependency
}

type GoDependency struct {
	Path       string
	Repository string
	Version    string
}

func ValidateGoMod(path string, content []byte) error {
	_, err := ParseGoMod(path, content)
	return err
}

func ParseGoMod(filePath string, content []byte) (GoModule, error) {
	file, err := modfile.Parse(filePath, content, nil)
	if err != nil {
		return GoModule{}, fmt.Errorf("release set: parse %s: %w", filePath, err)
	}
	if len(file.Replace) != 0 {
		return GoModule{}, fmt.Errorf("release set: %s contains replace directives", filePath)
	}
	if file.Module == nil {
		return GoModule{}, errors.New("release set: go.mod has no module directive")
	}

	result := GoModule{
		Path:       file.Module.Mod.Path,
		Repository: pawnkitRepository(file.Module.Mod.Path),
	}
	for _, requirement := range file.Require {
		if !strings.HasPrefix(requirement.Mod.Path, "github.com/pawnkit/") {
			continue
		}
		version := requirement.Mod.Version
		if module.IsPseudoVersion(version) || !versionPattern.MatchString(version) {
			return GoModule{}, fmt.Errorf(
				"release set: %s uses unpublished PawnKit version %q",
				filePath,
				version,
			)
		}
		result.Dependencies = append(result.Dependencies, GoDependency{
			Path:       requirement.Mod.Path,
			Repository: pawnkitRepository(requirement.Mod.Path),
			Version:    version,
		})
	}
	sort.Slice(result.Dependencies, func(i, j int) bool {
		return result.Dependencies[i].Path < result.Dependencies[j].Path
	})
	return result, nil
}

func ValidateGoModuleGraph(modules []GoModule) error {
	byRepository := make(map[string]GoModule, len(modules))
	for _, current := range modules {
		if current.Repository == "" {
			continue
		}
		if _, ok := dependencyLayer[current.Repository]; !ok {
			return fmt.Errorf("release set: unclassified PawnKit repository %q", current.Repository)
		}
		if _, exists := byRepository[current.Repository]; exists {
			return fmt.Errorf("release set: duplicate module for %q", current.Repository)
		}
		byRepository[current.Repository] = current
	}

	for _, current := range modules {
		if current.Repository == "pawn-corpus" {
			continue
		}
		currentLayer, currentKnown := dependencyLayer[current.Repository]
		for _, dependency := range current.Dependencies {
			if dependency.Repository == "pawn-corpus" {
				return fmt.Errorf(
					"release set: production module %s depends on pawn-corpus",
					current.Repository,
				)
			}
			dependencyLayerValue, dependencyKnown := dependencyLayer[dependency.Repository]
			if !dependencyKnown {
				return fmt.Errorf("release set: unclassified PawnKit repository %q", dependency.Repository)
			}
			if currentKnown && dependencyLayerValue > currentLayer {
				return fmt.Errorf(
					"release set: reversed dependency %s -> %s",
					current.Repository,
					dependency.Repository,
				)
			}
		}
	}

	state := make(map[string]uint8, len(byRepository))
	var visit func(string) error
	visit = func(repository string) error {
		switch state[repository] {
		case 1:
			return fmt.Errorf("release set: dependency cycle contains %s", repository)
		case 2:
			return nil
		}
		state[repository] = 1
		for _, dependency := range byRepository[repository].Dependencies {
			if _, included := byRepository[dependency.Repository]; included {
				if err := visit(dependency.Repository); err != nil {
					return err
				}
			}
		}
		state[repository] = 2
		return nil
	}
	for repository := range byRepository {
		if err := visit(repository); err != nil {
			return err
		}
	}
	return nil
}

func pawnkitRepository(modulePath string) string {
	const prefix = "github.com/pawnkit/"
	if !strings.HasPrefix(modulePath, prefix) {
		return ""
	}
	return path.Base(strings.TrimPrefix(modulePath, prefix))
}

var dependencyLayer = map[string]int{
	"pawnkit-core":     0,
	"pawn-corpus":      0,
	"goamx":            1,
	"pawn-parser":      1,
	"pawn-project":     1,
	"tree-sitter-pawn": 1,
	"pawn-analysis":    2,
	"pawn-api":         2,
	"pawn-plugin-host": 3,
	"pawndebug":        3,
	"pawndoc":          3,
	"pawnfmt":          3,
	"pawnlint":         3,
	"pawnmigrate":      3,
	"pawntest":         3,
	"pawn-actions":     4,
	"pawnkit-cli":      4,
	"pawnlsp":          4,
	"pawnserver":       4,
	"vscode-pawn":      4,
	"pawnkit.dev":      5,
}
