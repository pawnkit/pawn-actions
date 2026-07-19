package actions_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAMLFilesParse(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".yml" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var document yaml.Node
		if err := yaml.Unmarshal(content, &document); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestActionMetadataHasCompositeRuns(t *testing.T) {
	for _, directory := range []string{"setup", "check", "fmt", "lint", "test"} {
		content, err := os.ReadFile(filepath.Join(directory, "action.yml"))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := yaml.Unmarshal(content, &value); err != nil {
			t.Fatal(err)
		}
		runs, ok := value["runs"].(map[string]any)
		if !ok || runs["using"] != "composite" {
			t.Fatalf("%s action has invalid runs metadata", directory)
		}
	}
}

func TestThirdPartyActionsUseFullCommitPins(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".yml" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for line := range strings.SplitSeq(string(content), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "uses:") && !strings.HasPrefix(line, "- uses:") {
				continue
			}
			target := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "- "), "uses:"))
			if strings.HasPrefix(target, "pawnkit/") {
				continue
			}
			_, ref, ok := strings.Cut(target, "@")
			if !ok || len(ref) != 40 || strings.Trim(ref, "0123456789abcdef") != "" {
				t.Errorf("%s has unpinned action %q", path, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowTemplatePropertiesParse(t *testing.T) {
	content, err := os.ReadFile(".github/workflow-templates/pawn-check.properties.json")
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil || value["name"] == "" {
		t.Fatalf("properties error=%v value=%v", err, value)
	}
}
