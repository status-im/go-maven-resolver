package main

import (
	"fmt"
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

type Project struct {
	GroupId      string       `xml:"groupId"`
	ArtifactId   string       `xml:"artifactId"`
	Name         string       `xml:"name"`
	Description  string       `xml:"description"`
	Version      string       `xml:"version"`
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
	Versioning Versioning `xml:"versioning"`
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

func (d *Dependency) HasVersion() bool {
	return d.Version != "" && !strings.HasPrefix(d.Version, "${")
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
		d.GroupIdAsPath(), d.ArtifactId, d.Version, d.ArtifactId, d.Version)
}
