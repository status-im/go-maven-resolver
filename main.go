package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Optional   bool   `xml:"optional"`
}

type Project struct {
	ArtifactId   string       `xml:"artifactId"`
	Name         string       `xml:"name"`
	Description  string       `xml:"description"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

func readPOM(path string) (*Project, error) {
	var project Project
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	bytes, _ := ioutil.ReadAll(file)
	xml.Unmarshal(bytes, &project)
	return &project, nil
}

var pomPath string

func flagsInit() {
	flag.StringVar(&pomPath, "path", "", "Path to the POM file to read")
	flag.Parse()
}

func main() {
	flagsInit()

	if pomPath == "" {
		fmt.Println("POM file path not specified!")
		os.Exit(1)
	}

	x, _ := readPOM(pomPath)
	fmt.Println(x)
}
