package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func parsePOM(bytes []byte) *Project {
	var project Project
	xml.Unmarshal(bytes, &project)
	return &project
}

func parseMeta(bytes []byte) *Metadata {
	var meta Metadata
	xml.Unmarshal(bytes, &meta)
	return &meta
}

func readPOM(path string) (*Project, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return parsePOM(bytes), nil
}

/* TODO implement a timeout */
func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(
			fmt.Sprintf("failed to fetch with: %d", resp.StatusCode))
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func repos() []string {
	return []string{
		"https://repo.maven.apache.org/maven2",
		"https://dl.google.com/dl/android/maven2",
		"https://repository.sonatype.org/content/groups/sonatype-public-grid",
		"https://plugins.gradle.org/m2",
		"https://maven.java.net/content/repositories/releases",
		"https://jcenter.bintray.com",
		"https://jitpack.io",
		"https://repo1.maven.org/maven2",
	}
}

func tryRepos(path string) (string, []byte, error) {
	for _, repo := range repos() {
		url := repo + "/" + path
		project, err := fetch(url)
		if err == nil {
			return url, project, nil
		} else {
			return "", nil, err
		}
	}
	return "", nil, errors.New("unable to find url")
}
