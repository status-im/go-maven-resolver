package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/status-im/go-maven-resolver/fetcher"
)

var l *log.Logger

var workersNum int
var requestTimeout int
var reposFile string
var ignoreScopes string
var recursive bool

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
			strings.Join(fetcher.DefaultRepos, "\n"))
		defaultUsage()
	}

	flag.BoolVar(&recursive, "recursive", true, "Should recursive resolution be done")
	flag.IntVar(&workersNum, "workers", 50, "Number of fetching workers.")
	flag.IntVar(&requestTimeout, "timeout", 2, "HTTP request timeout in seconds.")
	flag.StringVar(&reposFile, "reposFile", "", "Path file with repo URLs to check.")
	flag.StringVar(&ignoreScopes, "ignoreScopes", "provided,system,test", "Scopes to ignore.")
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

	/* Manages traversal threads, which go through the tree of dependencies
	 * And spawn new Go routines for each new node in the tree. */
	finder := Finder{
		deps:         make(map[string]bool),
		fetchers:     fetcher.NewPool(workersNum, requestTimeout, repos),
		ignoreScopes: strings.Split(ignoreScopes, ","),
		recursive:    recursive,
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
