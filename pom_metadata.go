package main

import (
	"encoding/xml"
)

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

func MetadataFromBytes(bytes []byte) Metadata {
	var meta Metadata
	xml.Unmarshal(bytes, &meta)
	return meta
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
