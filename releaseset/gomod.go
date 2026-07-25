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
	Version      string
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
	byKey := make(map[string]GoModule, len(modules))
	keysByRepository := make(map[string][]string, len(modules))
	for _, current := range modules {
		if current.Repository == "" {
			continue
		}
		if _, ok := dependencyLayer[current.Repository]; !ok {
			return fmt.Errorf("release set: unclassified PawnKit repository %q", current.Repository)
		}
		key := moduleKey(current.Repository, current.Version)
		if _, exists := byKey[key]; exists {
			return fmt.Errorf("release set: duplicate module for %q", key)
		}
		byKey[key] = current
		keysByRepository[current.Repository] = append(keysByRepository[current.Repository], key)
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

	state := make(map[string]uint8, len(byKey))
	var visit func(string) error
	visit = func(key string) error {
		switch state[key] {
		case 1:
			return fmt.Errorf("release set: dependency cycle contains %s", key)
		case 2:
			return nil
		}
		state[key] = 1
		for _, dependency := range byKey[key].Dependencies {
			dependencyKey := moduleKey(dependency.Repository, dependency.Version)
			if _, included := byKey[dependencyKey]; !included &&
				len(keysByRepository[dependency.Repository]) == 1 {
				dependencyKey = keysByRepository[dependency.Repository][0]
			}
			if _, included := byKey[dependencyKey]; included {
				if err := visit(dependencyKey); err != nil {
					return err
				}
			}
		}
		state[key] = 2
		return nil
	}
	for key := range byKey {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

func moduleKey(repository string, version string) string {
	if version == "" {
		return repository
	}
	return repository + "@" + version
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
