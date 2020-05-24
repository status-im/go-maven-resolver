package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

var workersNum int
var requestTimeout int
var reposFile string
var ignoreScopes string

var helpMessage string = `
This tool takes a names of a Java Maven packages
via STDIN and returns the URLs of all its dependencies.

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
