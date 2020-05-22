package main

import (
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
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

func MetadataFromBytes(reader io.ReadCloser) (*Metadata, error) {
	defer reader.Close()
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	var meta Metadata
	err := decoder.Decode(&meta)
	return &meta, err
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
