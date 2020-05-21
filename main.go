package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   bool   `xml:"optional"`
}

type Project struct {
	GroupId      string       `xml:"groupId"`
	ArtifactId   string       `xml:"artifactId"`
	Name         string       `xml:"name"`
	Description  string       `xml:"description"`
	Version      string       `xml:"version"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
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

func (d *Dependency) GetPath() string {
	path := strings.ReplaceAll(d.GroupId, ".", "/")
	return fmt.Sprintf("%s/%s/%s/%s-%s.pom",
		path, d.ArtifactId, d.Version, d.ArtifactId, d.Version)
}

func parsePOM(bytes []byte) *Project {
	var project Project
	xml.Unmarshal(bytes, &project)
	return &project
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

func fetchPOM(url string) (*Project, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch")
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parsePOM(bytes), nil
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

func tryRepos(dep Dependency) (string, *Project, error) {
	for _, repo := range repos() {
		var url string = repo + "/" + dep.GetPath()
		project, err := fetchPOM(url)
		if err == nil {
			return url, project, nil
		}
	}
	return "", nil, errors.New("unable to find URL")
}

func findAllUrls(dep Dependency) (urls []string, err error) {
	url, project, err := tryRepos(dep)
	fmt.Println("Found:", url)
	if err == nil {
		for _, subDep := range project.Dependencies {
			urls, err = findAllUrls(subDep)
			if err != nil {
				fmt.Println("failed to find:", err)
				return nil, err
			}
		}
	}
	return append(urls, url), nil
}

var packageName string
var pomPath string

func flagsInit() {
	flag.StringVar(&pomPath, "path", "", "Path to the POM file to read")
	flag.StringVar(&packageName, "name", "", "Name of Java package")
	flag.Parse()
}

func main() {
	flagsInit()

	if pomPath == "" && packageName == "" {
		fmt.Println("POM file path or package name not specified!")
		os.Exit(1)
	}

	if pomPath != "" {
		project, err := readPOM(pomPath)
		if err != nil {
			fmt.Println("failed to read file:", err)
			os.Exit(1)
		}
		fmt.Println(project)
	} else if packageName != "" {
		dep := DependencyFromString(packageName)
		urls, err := findAllUrls(*dep)
		if err != nil {
			fmt.Println("failed to find URL:", err)
			os.Exit(1)
		}
		fmt.Println(urls)
	}
}
