package releaseset

import "fmt"

// SelectArtifact returns one component artifact for a tested target.
func SelectArtifact(set Set, componentName, target string) (Component, Artifact, error) {
	if err := set.Validate(); err != nil {
		return Component{}, Artifact{}, err
	}
	for _, component := range set.Components {
		if component.Name != componentName {
			continue
		}
		for _, artifact := range component.Artifacts {
			if artifact.Target == target {
				return component, artifact, nil
			}
		}
		return Component{}, Artifact{}, fmt.Errorf(
			"release set: component %q has no artifact for %q",
			componentName,
			target,
		)
	}
	return Component{}, Artifact{}, fmt.Errorf("release set: component %q was not found", componentName)
}
