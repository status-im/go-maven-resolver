# Description

This tool takes a names of a Java packages and returns the URLs of POMs for all its dependencies.

# Usage

In basic terms the tool takes Maven formatted package names via `STDIN` and outputs URLs for all dependencies via `STDOUT`:
```
 $ echo commons-io:commons-io:2.4 | ./go-maven-resolver
https://repo.maven.apache.org/maven2/commons-io/commons-io/2.4/commons-io-2.4.pom
https://repo.maven.apache.org/maven2/org/apache/commons/commons-parent/25/commons-parent-25.pom
https://repo.maven.apache.org/maven2/org/apache/apache/9/apache-9.pom
```
The package name takes the Maven format: `<groupId>:<artifactId>:<version>`

There's also a few flags available:
```
Usage of ./go-maven-resolver:
  -exitCode
    	Set exit code on any resolving failures. (default true)
  -ignoreOptional
    	Ignore optional dependencies. (default true)
  -ignoreScopes string
    	Scopes to ignore. (default "provided,system,test")
  -recursive
    	Should recursive resolution be done (default true)
  -reposFile string
    	Path file with repo URLs to check.
  -retries int
    	HTTP request retries on non-404 codes. (default 2)
  -timeout int
    	HTTP request timeout in seconds. (default 2)
  -workers int
    	Number of fetching workers. (default 50)
```

# Details

The way this works is, for each given package name:

1. Iterates through a list of Maven repositories
2. Finds and downloads the POM for the given package
2. Analyzes the POM to find its dependencies
3. Repeats the process recursively until all POMs are found

The fetching is done by a pool of workers to avoid running out of sockets.

# Reasoning

I've decided to write this because I could not find a way to achieve the same thing using Gradle or Maven.
Using commands like `mvn dependency:list` or `mvn help:effective-pom` is too slow and give hard to parse output. It also attempts to download all of the necessary JARs and POMs which is unnecessary.

This is used to generate data for managing dependencies for a Gradle project via [Nix package manager](https://nixos.org/nix/) in our [status-react](https://github.com/status-im/status-react/tree/develop/nix/deps/gradle) repo.

For more details see [this Gradle Forum post](https://discuss.gradle.org/t/how-to-get-full-list-of-dependencies-and-their-meta/35825).
