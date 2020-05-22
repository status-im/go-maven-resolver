package main

import (
	"fmt"
	"sort"
	"strings"
)

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   bool   `xml:"optional"`

	/* This will be set by fetching code */
	URL string
}

type Parent struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type Project struct {
	GroupId      string       `xml:"groupId"`
	ArtifactId   string       `xml:"artifactId"`
	Name         string       `xml:"name"`
	Version      string       `xml:"version"`
	Parent       Parent       `xml:"parent"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

type Versioning struct {
	Latest   string   `xml:"latest"`
	Release  string   `xml:"release"`
	Versions []string `xml:"versions>version"`
}

type Metadata struct {
	GroupId    string     `xml:"groupId"`
	ArtifactId string     `xml:"artifactId"`
	Version    string     `xml:"version"`
	Versioning Versioning `xml:"versioning"`
}

/* Sometimes groupId is not specified in project */
func (p Project) GetGroupId() string {
	if p.GroupId != "" {
		return p.GroupId
	} else {
		return p.Parent.GroupId
	}
}

/* Maven POM file fields can reference project */
func (p Project) GetDependencies() []Dependency {
	var deps []Dependency
	for _, dep := range p.Dependencies {
		if dep.GroupId == "${project.groupId}" {
			dep.GroupId = p.GetGroupId()
		}
		if dep.GroupId == "${pom.groupId}" {
			dep.GroupId = p.GetGroupId()
		}
		if dep.Version == "${project.version}" {
			dep.Version = p.Version
		}
		deps = append(deps, dep)
	}
	return deps
}

func DependencyFromString(data string) *Dependency {
	tokens := strings.Split(data, ":")
	if len(tokens) < 3 {
		return nil
	}
	return &Dependency{
		GroupId:    tokens[0],
		ArtifactId: tokens[1],
		Version:    tokens[2],
	}
}

func (d Dependency) String() string {
	return fmt.Sprintf("<Dep G=%s A=%s V=%s O=%t S=%s >",
		d.GroupId, d.ArtifactId, d.Version, d.Optional, d.Scope)
}

func (d *Dependency) HasVersion() bool {
	return d.Version != "" && d.Version != "unspecified" && !strings.HasPrefix(d.Version, "${")
}

func (d *Dependency) IsProvided() bool {
	return d.Scope == "provided"
}

func (d *Dependency) IsSystem() bool {
	return d.Scope == "system"
}

/* version strings can be tricky, like "[2.1.0,2.1.1]" */
func (d *Dependency) GetVersion() string {
	clean := strings.Trim(d.Version, "[]")
	tokens := strings.Split(clean, ",")
	sort.Strings(tokens)
	return tokens[len(tokens)-1]
}

func (d *Dependency) GroupIdAsPath() string {
	return strings.ReplaceAll(d.GroupId, ".", "/")
}

func (d *Dependency) GetMetaPath() string {
	return fmt.Sprintf("%s/%s/maven-metadata.xml",
		d.GroupIdAsPath(), d.ArtifactId)
}

func (d *Dependency) GetPOMPath() string {
	return fmt.Sprintf("%s/%s/%s/%s-%s.pom",
		d.GroupIdAsPath(), d.ArtifactId,
		d.GetVersion(), d.ArtifactId, d.GetVersion())
}

func (m *Metadata) GetLatest() string {
	if m.Versioning.Latest != "" {
		return m.Versioning.Latest
	} else if m.Versioning.Release != "" {
		return m.Versioning.Release
	} else {
		return m.Version
	}
}
