package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/status-im/go-maven-resolver/fetcher"
	"github.com/status-im/go-maven-resolver/finder"
	"github.com/status-im/go-maven-resolver/pom"
)

var l *log.Logger

var (
	workersNum     int
	requestRetries int
	requestTimeout int
	reposFile      string
	ignoreScopes   string
	ignoreOptional bool
	recursive      bool
)

const helpMessage string = `
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
		fmt.Fprintf(os.Stderr, // TODO unhandled error
			strings.Trim(helpMessage, "\t "),
			strings.Join(fetcher.DefaultRepos, "\n"))
		defaultUsage()
	}

	flag.BoolVar(&recursive, "recursive", true, "Should recursive resolution be done")
	flag.IntVar(&workersNum, "workers", 50, "Number of fetching workers.")
	flag.IntVar(&requestRetries, "retries", 2, "HTTP request retries on non-404 codes.")
	flag.IntVar(&requestTimeout, "timeout", 2, "HTTP request timeout in seconds.")
	flag.StringVar(&reposFile, "reposFile", "", "Path file with repo URLs to check.")
	flag.StringVar(&ignoreScopes, "ignoreScopes", "provided,system,test", "Scopes to ignore.")
	flag.BoolVar(&ignoreOptional, "ignoreOptional", true, "Ignore optional dependencies.")
	flag.Parse()
}

func main() {
	l = log.New(os.Stderr, "", log.Lshortfile)

	flagsInit()

	repos := fetcher.DefaultRepos

	if reposFile != "" {
		lines, err := ReadFileToList(reposFile)
		if err != nil {
			l.Println("failed to read repos file:", err)
			os.Exit(1)
		}
		repos = lines
	}

	/* Controls which dependencies are resolved. */
	finderOpts := finder.Options{
		IgnoreScopes:    strings.Split(ignoreScopes, ","),
		IgnoreOptional:  ignoreOptional,
		RecursiveSearch: recursive,
	}

	/* A separate pool of fetcher workers prevents running out of sockets */
	fch := fetcher.New(requestRetries, workersNum, requestTimeout, repos, l)

	/* Manages traversal threads, which go through the tree of dependencies
	 * And spawn new Go routines for each new node in the tree. */
	fnr := finder.New(finderOpts, fch, l)

	/* We read Maven formatted names of packages from STDIN. */
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dep, err := pom.DependencyFromString(scanner.Text())
		if err != nil {
			l.Printf("failed to parse input: %s", err)
			continue
		}
		/* The threads print found URLs into STDOUT. */
		fnr.Resolve(*dep)
	}

	/* Reading from STDIN might fail. */
	if err := scanner.Err(); err != nil {
		l.Printf("error:", err) // TODO unhandled error
	}

	/* Each FindUrls() call can spawn more recursive FindUrls() routines.
	 * To know when to stop the process they also increment the WaitGroup. */
	fnr.Wait()
}
