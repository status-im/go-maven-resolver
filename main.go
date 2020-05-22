package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
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

func fetch(url string) ([]byte, error) {
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
	return bytes, nil
}

func fetchPOM(url string) (*Project, error) {
	bytes, err := fetch(url)
	if err != nil {
		return nil, err
	}
	return parsePOM(bytes), nil
}

func fetchMeta(url string) (*Metadata, error) {
	bytes, err := fetch(url)
	if err != nil {
		return nil, err
	}
	return parseMeta(bytes), nil
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
		}
	}
	return "", nil, errors.New(fmt.Sprintf("unable to find URL: %s", path))
}

func resolveDep(dep Dependency) (string, *Project, error) {
	if !dep.HasVersion() {
		_, bytes, err := tryRepos(dep.GetMetaPath())
		if err != nil {
			fmt.Println("Meta err:", err)
			return "", nil, errors.New("no meta for dep")
		}
		meta := parseMeta(bytes)
		dep.Version = meta.Versioning.Latest
	}
	url, bytes, err := tryRepos(dep.GetPOMPath())
	if err != nil {
		return "", nil, err
	}
	return url, parsePOM(bytes), nil
}

type POMFinder struct {
	deps sync.Map
	wg   sync.WaitGroup
}

/* Threaded version */
func (f *POMFinder) FindUrlsT(dep Dependency) {
	defer f.wg.Done()

	if _, ok := f.deps.Load(dep); ok {
		return
	}

	url, project, err := resolveDep(dep)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if url == "" {
		fmt.Println("no URL found")
		return
	}

	fmt.Println("Found:", url)
	f.deps.Store(dep, url)

	for _, subDep := range project.Dependencies {
		f.wg.Add(1)
		go f.FindUrlsT(subDep)
	}
}

/* Recursive version */
func (f *POMFinder) FindUrlsR(dep Dependency) {
	if _, ok := f.deps.Load(dep); ok {
		return
	}

	url, project, err := resolveDep(dep)
	if err != nil {
		fmt.Println("Error:", err)
	}

	if url == "" {
		fmt.Println("no URLs found")
	}

	fmt.Println("Found:", url)
	f.deps.Store(dep, url)

	for _, subDep := range project.Dependencies {
		f.FindUrlsR(subDep)
	}
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
		f := POMFinder{}
		f.wg.Add(1)
		go f.FindUrlsT(*dep)
		f.wg.Wait()
	}
}
