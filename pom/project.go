package pom

import (
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
)

/* Root object in XML POM files defining packages. */
type Project struct {
	GroupId      string       `xml:"groupId"`
	ArtifactId   string       `xml:"artifactId"`
	Name         string       `xml:"name"`
	Version      string       `xml:"version"`
	Parent       Dependency   `xml:"parent"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
	Build        struct {
		Plugins []Dependency `xml:"plugins>plugin"`
	}
}

/* For reading Project from downloaded POM file. */
func ProjectFromReader(reader io.ReadCloser) (*Project, error) {
	defer reader.Close()
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	var project Project
	err := decoder.Decode(&project)
	return &project, err
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
	if p.Parent.ArtifactId != "" {
		deps = append(deps, p.Parent)
	}
	for _, dep := range p.Dependencies {
		deps = append(deps, dep.FixFields(p))
	}
	for _, dep := range p.Build.Plugins {
		deps = append(deps, dep.FixFields(p))
	}
	return deps
}
