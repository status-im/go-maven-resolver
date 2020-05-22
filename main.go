package main

import (
	"bufio"
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
		/* TODO could use found repo below */
		_, bytes, err := tryRepos(dep.GetMetaPath())
		if err != nil {
			fmt.Println("Meta err:", err, dep)
			return "", nil, errors.New("no meta for dep")
		}
		meta := parseMeta(bytes)
		dep.Version = meta.GetLatest()
	}
	url, bytes, err := tryRepos(dep.GetPOMPath())
	if err != nil {
		return "", nil, err
	}
	return url, parsePOM(bytes), nil
}

type POMFinder struct {
	deps sync.Map       /* to avoid checking the same dep */
	wg   sync.WaitGroup /* to figure out when it's done */
}

func (f *POMFinder) FindUrls(dep Dependency) {
	defer f.wg.Done()

	if _, ok := f.deps.Load(dep); !ok {
		f.deps.Store(dep, "checking")
	} else {
		return /* thread already working on it */
	}

	url, project, err := resolveDep(dep)
	if err != nil {
		fmt.Println("Error:", err, dep)
		return
	}

	if url == "" {
		fmt.Println("no URL found")
		return
	}

	fmt.Println(url)
	f.deps.Store(dep, url)

	for _, subDep := range project.Dependencies {
		if subDep.Optional || subDep.IsProvided() {
			continue
		}
		f.wg.Add(1)
		go f.FindUrls(subDep)
	}
}

var reposPath string

func flagsInit() {
	flag.StringVar(&reposPath, "repos", "", "Path file with repo URLs to check.")
	flag.Parse()
}

func main() {
	//flagsInit() TODO

	/* manages threads */
	f := POMFinder{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dep := DependencyFromString(scanner.Text())
		f.wg.Add(1)
		go f.FindUrls(*dep)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("STDIN err:", err)
	}

	f.wg.Wait()
}
