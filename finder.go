package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/status-im/go-maven-resolver/fetcher"
)

type Finder struct {
	deps         map[string]bool /* to avoid checking the same dep */
	mtx          sync.Mutex      /* for locking access to the deps map */
	wg           sync.WaitGroup  /* to figure out when it's done */
	fetchers     fetcher.Pool    /* pool of workers for HTTP requests */
	ignoreScopes []string        /* list of scopes to ignore */
	recursive    bool            /* recursive resolution control */
}

func (f *Finder) ResolveDep(dep Dependency) (string, *Project, error) {
	var rval *fetcher.Result
	var repo string
	result := make(chan *fetcher.Result)
	defer close(result)

	if !dep.HasVersion() {
		path := dep.GetMetaPath()
		/* We use workers for HTTP request to avoid running out of sockets */
		f.fetchers.Queue <- &fetcher.Job{result, path, repo}
		rval = <-result
		if rval.Url == "" {
			return "", nil, fmt.Errorf("no metadata found: %s", rval.Url)
		}
		meta, err := MetadataFromReader(rval.Data)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse: %s", rval.Url)
		}
		dep.Version = meta.GetLatest()
		/* This is to optimize the POM searching and avoid
		 * checking more repos than is necessary. */
		repo = rval.Repo
	}

	path := dep.GetPOMPath()
	/* We use workers for HTTP request to avoid running out of sockets */
	f.fetchers.Queue <- &fetcher.Job{result, path, repo}
	rval = <-result

	if rval.Data == nil {
		return "", nil, errors.New("no pom data")
	}
	project, err := ProjectFromReader(rval.Data)
	if err != nil {
		return "", nil, err
	}
	return rval.Url, project, nil
}

func (f *Finder) InvalidDep(dep Dependency) bool {
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
func (f *Finder) LockDep(dep Dependency) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	id := dep.ID()
	if f.deps[id] == true {
		return false
	}
	f.deps[id] = true
	return true
}

func (f *Finder) FindUrls(dep Dependency) {
	defer f.wg.Done()

	/* Check if the dependency is being checked or was already found. */
	if !f.LockDep(dep) {
		return
	}

	/* Does the job of finding the download URL for dependecy POM file. */
	url, project, err := f.ResolveDep(dep)
	if err != nil {
		l.Printf("error: '%s' for: %s", err, dep)
		return
	}

	/* This should never happen, since most of the time if ResolveDep()
	 * fails it is due to an HTTP error or XML parsing error. */
	if url == "" {
		l.Printf("error: 'no URL found' for: %s", dep)
		return
	}

	/* This is what shows the found URL in STDOUT. */
	fmt.Println(url)

	if !f.recursive {
		return
	}

	/* Now that we have the POM we can check all the sub-dependencies. */
	for _, subDep := range project.GetDependencies() {
		if f.InvalidDep(subDep) {
			continue
		}
		f.wg.Add(1)
		go f.FindUrls(subDep)
	}
}
