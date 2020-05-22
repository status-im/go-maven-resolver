package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
)

func resolveDep(dep Dependency) (string, *Project, error) {
	if !dep.HasVersion() {
		/* TODO could use found repo below */
		_, bytes, err := tryRepos(dep.GetMetaPath())
		if err != nil {
			return "", nil, errors.New("no metadata found")
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

func InvalidDep(dep Dependency) bool {
	return dep.Optional || dep.IsProvided() || dep.IsSystem()
}

type POMFinder struct {
	deps map[Dependency]bool /* to avoid checking the same dep */
	mtx  sync.Mutex          /* for locking access to the deps map */
	wg   sync.WaitGroup      /* to figure out when it's done */
}

func (f *POMFinder) LockDep(dep Dependency) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.deps[dep] == true {
		return false
	}
	f.deps[dep] = true
	return true
}

/* TODO use a worker pool */
func (f *POMFinder) FindUrls(dep Dependency) {
	defer f.wg.Done()

	if !f.LockDep(dep) {
		return
	}

	url, project, err := resolveDep(dep)
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
		go f.FindUrls(subDep)
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

	/* manages threads */
	f := POMFinder{deps: make(map[Dependency]bool)}

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
