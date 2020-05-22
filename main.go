package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
)

func resolveDep(dep Dependency, fetchers FetcherPool) (string, *Project, error) {
	var rval FetcherResult
	var repo string
	result := make(chan FetcherResult)
	defer close(result)

	if !dep.HasVersion() {
		/* TODO could use found repo below */
		path := dep.GetMetaPath()
		fetchers.queue <- FetcherJob{result, path, repo}
		rval = <-result
		if rval.data == nil {
			return "", nil, errors.New("no metadata found")
		}
		meta := parseMeta(rval.data)
		dep.Version = meta.GetLatest()
		repo = rval.repo
	}

	path := dep.GetPOMPath()
	fetchers.queue <- FetcherJob{result, path, repo}
	rval = <-result

	if rval.data == nil {
		return "", nil, errors.New("no POM found")
	}
	return rval.url, parsePOM(rval.data), nil
}

func InvalidDep(dep Dependency) bool {
	return dep.Optional || dep.IsProvided() || dep.IsSystem()
}

type POMFinder struct {
	deps map[string]bool /* to avoid checking the same dep */
	mtx  sync.Mutex      /* for locking access to the deps map */
	wg   sync.WaitGroup  /* to figure out when it's done */
}

func (f *POMFinder) LockDep(dep Dependency) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	id := dep.ID()
	if f.deps[id] == true {
		return false
	}
	f.deps[id] = true
	return true
}

/* TODO use a worker pool */
func (f *POMFinder) FindUrls(dep Dependency, fetchers FetcherPool) {
	defer f.wg.Done()

	found := f.LockDep(dep)
	if !found {
		return
	}

	url, project, err := resolveDep(dep, fetchers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err, dep)
		return
	}

	if url == "" {
		fmt.Println("no URL found")
		return
	}

	fmt.Println(url)

	for _, subDep := range project.GetDependencies() {
		if InvalidDep(subDep) {
			continue
		}
		f.wg.Add(1)
		go f.FindUrls(subDep, fetchers)
	}
}

var javaVersion string
var reposPath string

func flagsInit() {
	flag.StringVar(&reposPath, "repos", "", "Path file with repo URLs to check.")
	flag.StringVar(&javaVersion, "repos", "", "Path file with repo URLs to check.")
	flag.Parse()
}

func main() {
	//flagsInit() TODO

	/* manages traversal threads */
	f := POMFinder{deps: make(map[string]bool)}

	/* managed fetcher threads */
	p := NewFetcherPool(10)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dep := DependencyFromString(scanner.Text())
		f.wg.Add(1)
		go f.FindUrls(*dep, p)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	f.wg.Wait()
}
