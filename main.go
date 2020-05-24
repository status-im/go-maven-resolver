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
		path := dep.GetMetaPath()
		/* We use workers for HTTP request to avoid running out of sockets */
		f.fetchers.queue <- FetcherJob{result, path, repo}
		rval = <-result
		if rval.data == nil {
			return "", nil, errors.New("no metadata found")
		}
		meta, err := MetadataFromReader(rval.data)
		if err != nil {
			return "", nil, err
		}
		dep.Version = meta.GetLatest()
		/* This is to optimize the POM searching and avoid
		 * checking more repos than is necessary. */
		repo = rval.repo
	}

	path := dep.GetPOMPath()
	/* We use workers for HTTP request to avoid running out of sockets */
	f.fetchers.queue <- FetcherJob{result, path, repo}
	rval = <-result

	if rval.data == nil {
		return "", nil, errors.New("no POM found")
	}
	project, err := ProjectFromReader(rval.data)
	if err != nil {
		return "", nil, err
	}
	return rval.url, project, nil
}

func (f *POMFinder) InvalidDep(dep Dependency) bool {
	/* Check if the scope matches any of the ignored ones. */
	for i := range f.ignoreScopes {
		if dep.Scope == f.ignoreScopes[i] {
			return true
		}
	}
	/* Else just check if it's optional, TODO parametrize. */
	return dep.Optional
}

/* We use a map of dependency IDs to avoid repeating a search. */
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

func (f *POMFinder) FindUrls(dep Dependency) {
	defer f.wg.Done()

	/* Check if the dependency is being checked or was already found. */
	if !f.LockDep(dep) {
		return
	}

	/* Does the job of finding the download URL for dependecy POM file. */
	url, project, err := f.ResolveDep(dep)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err, dep)
		return
	}

	/* This should never happen, since most of the time if ResolveDep()
	 * fails it is due to an HTTP error or XML parsing error. */
	if url == "" {
		fmt.Fprintln(os.Stderr, "no URL found", dep)
		return
	}

	/* This is what shows the found URL in STDOUT. */
	fmt.Println(url)

	/* Now that we have the POM we can check all the sub-dependencies. */
	for _, subDep := range project.GetDependencies() {
		if f.InvalidDep(subDep) {
			continue
		}
		f.wg.Add(1)
		go f.FindUrls(subDep)
	}
}

var workersNum int
var requestTimeout int
var reposFile string
var ignoreScopes string

var helpMessage string = `
This is a tool that takes a name of a Java Maven package
or a POM file and returns the URLs of all its dependencies.

echo commons-io:commons-io:2.4 | ./go-maven-resolver

The default repos used for searching are:
%s

You can provide your own list using the -reposFile flag.

`

func flagsInit() {
	defaultUsage := flag.Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			strings.Trim(helpMessage, "\t "),
			strings.Join(defaultRepos(), "\n"))
		defaultUsage()
	}

	flag.IntVar(&workersNum, "workers", 50, "Number of fetching workers.")
	flag.IntVar(&requestTimeout, "timeout", 2, "HTTP request timeout in seconds.")
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

	/* Manages traversal threads, which go through the tree of dependencies
	 * And spawn new Go routines for each new node in the tree. */
	finder := POMFinder{
		deps:         make(map[string]bool),
		fetchers:     NewFetcherPool(workersNum, requestTimeout, repos),
		ignoreScopes: strings.Split(ignoreScopes, ","),
	}

	/* We read Maven formatted names of packages from STDIN.
	 * The threads print found URLs into STDOUT. */
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dep := DependencyFromString(scanner.Text())
		finder.wg.Add(1)
		go finder.FindUrls(*dep)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	/* Each FindUrls() call can spawn more recursive FindUrls() routines.
	 * To know when to stop the process they also increment the WaitGroup. */
	finder.wg.Wait()
}
