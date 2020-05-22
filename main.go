package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
)

type POMFinder struct {
	deps         map[string]bool /* to avoid checking the same dep */
	mtx          sync.Mutex      /* for locking access to the deps map */
	wg           sync.WaitGroup  /* to figure out when it's done */
	fetchers     FetcherPool     /* pool of workers for HTTP requests */
	ignoreScopes []string        /* list of scopes to ignore */
}

func (f *POMFinder) ResolveDep(dep Dependency) (string, *Project, error) {
	var rval FetcherResult
	var repo string
	result := make(chan FetcherResult)
	defer close(result)

	if !dep.HasVersion() {
		/* TODO could use found repo below */
		path := dep.GetMetaPath()
		f.fetchers.queue <- FetcherJob{result, path, repo}
		rval = <-result
		if rval.data == nil {
			return "", nil, errors.New("no metadata found")
		}
		meta, err := MetadataFromBytes(rval.data)
		if err != nil {
			return "", nil, err
		}
		dep.Version = meta.GetLatest()
		repo = rval.repo
	}

	path := dep.GetPOMPath()
	f.fetchers.queue <- FetcherJob{result, path, repo}
	rval = <-result

	if rval.data == nil {
		return "", nil, errors.New("no POM found")
	}
	project, err := ProjectFromBytes(rval.data)
	if err != nil {
		return "", nil, err
	}
	return rval.url, project, nil
}

func (f *POMFinder) InvalidDep(dep Dependency) bool {
	/* check if the scope matches any of the ignored ones */
	for i := range f.ignoreScopes {
		if dep.Scope == f.ignoreScopes[i] {
			return true
		}
	}
	/* else just check if it's optional */
	return dep.Optional
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
func (f *POMFinder) FindUrls(dep Dependency) {
	defer f.wg.Done()

	if !f.LockDep(dep) {
		return
	}

	url, project, err := f.ResolveDep(dep)
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
		if f.InvalidDep(subDep) {
			continue
		}
		f.wg.Add(1)
		go f.FindUrls(subDep)
	}
}

var workersNum int
var reposFile string
var ignoreScopes string

func flagsInit() {
	flag.IntVar(&workersNum, "workers", 50, "Number of fetching workers.")
	flag.StringVar(&reposFile, "reposFile", "", "Path file with repo URLs to check.")
	flag.StringVar(&ignoreScopes, "ignoreScopes", "provided,system,test", "Scopes to ignore.")
	flag.Parse()
}

func main() {
	flagsInit()

	repos := defaultRepos()

	if reposFile != "" {
		lines, err := ReadFileToList(reposFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to read repos file:", err)
			os.Exit(1)
		}
		repos = lines
	}

	/* manages traversal threads */
	finder := POMFinder{
		deps:         make(map[string]bool),
		fetchers:     NewFetcherPool(workersNum, repos),
		ignoreScopes: strings.Split(ignoreScopes, ","),
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dep := DependencyFromString(scanner.Text())
		finder.wg.Add(1)
		go finder.FindUrls(*dep)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	finder.wg.Wait()
}
