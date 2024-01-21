package finder

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/go-maven-resolver/fetcher"
	"github.com/status-im/go-maven-resolver/pom"
)

type Options struct {
	IgnoreScopes     []string /* list of dependency scopes to ignore */
	IgnoreOptional   bool     /* if optional dependencies should be ignored */
	IgnoreTransitive bool     /* managed dependencies can be often ignored  */
	RecursiveSearch  bool     /* recursive dependency resolution switch */
}

type Finder struct {
	failed   bool            /* set to true if any requests failed */
	opts     Options         /* options for handling dependencies */
	fetchers fetcher.Fetcher /* pool of workers for HTTP requests */
	l        *log.Logger     /* for logging events */

	deps map[string]bool /* to avoid checking the same dep */
	mtx  sync.Mutex      /* for locking access to the deps map */
	wg   sync.WaitGroup  /* to figure out when it's done */
}

func New(opts Options, fetchers fetcher.Fetcher, logger *log.Logger) Finder {
	return Finder{
		deps:     make(map[string]bool),
		opts:     opts,
		fetchers: fetchers,
		l:        logger,
	}
}

func (f *Finder) ResolveDep(dep pom.Dependency) (string, *pom.Project, error) {
	var rval *fetcher.Result
	var repo string
	result := make(chan *fetcher.Result)
	defer close(result)

	// Add constants for retry logic
	const maxRetries = 3
	const retryDelay = time.Second * 1

	if !dep.HasVersion() {
		path := dep.GetMetaPath()
		/* We use workers for HTTP request to avoid running out of sockets */
		f.fetchers.Queue <- fetcher.NewJob(result, path, repo)
		rval = <-result
		if rval.Url == "" {
			return "", nil, fmt.Errorf("no metadata found: %s", rval.Url)
		}
		meta, err := pom.MetadataFromReader(rval.Data)
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
	// Retry logic for fetching data
	for attempt := 0; attempt < maxRetries; attempt++ {
		f.fetchers.Queue <- fetcher.NewJob(result, path, repo)
		rval = <-result

		if rval.Data != nil {
			break
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if rval.Data == nil {
		return "", nil, errors.New("no pom data")
	}

	project, err := pom.ProjectFromReader(rval.Data)
	if err != nil {
		return "", nil, err
	}
	return rval.Url, project, nil
}

func (f *Finder) InvalidDep(dep pom.Dependency) bool {
	if dep.Transitive {
		if f.opts.IgnoreTransitive {
			return true
		} else if dep.Scope == "none" {
			/* Unscoped transitive deps are mostly useless trash. */
			return true
		}
	}
	/* Check if the scope matches any of the ignored ones. */
	for i := range f.opts.IgnoreScopes {
		if dep.Scope == f.opts.IgnoreScopes[i] {
			return true
		}
	}
	/* Else just check if it's optional. */
	return f.opts.IgnoreOptional && dep.Optional
}

/* We use a map of dependency IDs to avoid repeating a search. */
func (f *Finder) LockDep(dep pom.Dependency) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	id := dep.ID()
	if f.deps[id] == true {
		return false
	}
	f.deps[id] = true
	return true
}

func (f *Finder) FindUrls(dep pom.Dependency) {
	defer f.wg.Done()

	/* Check if the dependency is being checked or was already found. */
	if !f.LockDep(dep) {
		return
	}

	/* Does the job of finding the download URL for dependecy POM file. */
	url, project, err := f.ResolveDep(dep)
	if err != nil {
		f.l.Printf("error: '%s' for: %s", err, dep)
		f.failed = true
		return
	}

	/* This should never happen, since most of the time if ResolveDep()
	 * fails it is due to an HTTP error or XML parsing error. */
	if url == "" {
		f.l.Printf("error: 'no URL found' for: %s", dep)
		f.failed = true
		return
	}

	/* This is what shows the found URL in STDOUT. */
	fmt.Println(url)

	if !f.opts.RecursiveSearch {
		return
	}

	/* Now that we have the POM we can check all the sub-dependencies. */
	for _, subDep := range project.GetDependencies() {
		if f.InvalidDep(subDep) {
			continue
		}
		f.Resolve(subDep)
	}
}

func (f *Finder) Resolve(dep pom.Dependency) {
	f.wg.Add(1)
	go f.FindUrls(dep)
}

func (f *Finder) Wait() {
	f.wg.Wait()
}

func (f *Finder) Failed() bool {
	return f.failed
}
