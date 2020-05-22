package main

import (
	"encoding/xml"
)

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

func ProjectFromBytes(bytes []byte) Project {
	var project Project
	xml.Unmarshal(bytes, &project)
	return project
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
