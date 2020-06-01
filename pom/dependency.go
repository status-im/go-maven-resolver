package pom

import (
	"fmt"
	"sort"
	"strings"
)

/* Dependency found in POM files. */
type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   bool   `xml:"optional"`
}

/* Maven uses a special format for dependency identifiers:
 *  - <groupId>:<artifactId>:<version> */
func DependencyFromString(data string) (*Dependency, error) {
	tokens := strings.Split(data, ":")
	if len(tokens) < 3 {
		return nil, fmt.Errorf("not a valid maven dependency: %s", data)
	}
	return &Dependency{
		GroupId:    tokens[0],
		ArtifactId: tokens[1],
		Version:    tokens[2],
	}, nil
}

/* POM file dependency fields can reference parent fields. */
func (d Dependency) FixFields(parent Project) Dependency {
	if d.GroupId == "${project.groupId}" {
		d.GroupId = parent.GetGroupId()
	}
	if d.GroupId == "${pom.groupId}" {
		d.GroupId = parent.GetGroupId()
	}
	if d.Version == "${project.version}" {
		d.Version = parent.Version
	}
	return d
}

func (d Dependency) ID() string {
	return fmt.Sprintf("%s:%s:%s", d.GroupId, d.ArtifactId, d.Version)
}

func (d Dependency) String() string {
	return fmt.Sprintf("<Dep ID=%s:%s:%s O=%t S=%s >",
		d.GroupId, d.ArtifactId, d.Version, d.Optional, d.Scope)
}

/* TODO this might need adjusting for cases with '${}' values */
func (d *Dependency) HasVersion() bool {
	return d.Version != "" && d.Version != "unspecified" && !strings.HasPrefix(d.Version, "${")
}

/* version strings can be tricky, like "[2.1.0,2.1.1]" */
func (d *Dependency) GetVersion() string {
	clean := strings.Trim(d.Version, "[]")
	tokens := strings.Split(clean, ",")
	sort.Strings(tokens)
	return tokens[len(tokens)-1]
}

/* Group ID in POM paths are split into subfolders. */
func (d *Dependency) GroupIdAsPath() string {
	return strings.ReplaceAll(d.GroupId, ".", "/")
}

/* For making URL paths for package metadata files */
func (d *Dependency) GetMetaPath() string {
	return fmt.Sprintf("%s/%s/maven-metadata.xml",
		d.GroupIdAsPath(), d.ArtifactId)
}

/* For making URL paths for POM files. */
func (d *Dependency) GetPOMPath() string {
	return fmt.Sprintf("%s/%s/%s/%s-%s.pom",
		d.GroupIdAsPath(), d.ArtifactId,
		d.GetVersion(), d.ArtifactId, d.GetVersion())
}
