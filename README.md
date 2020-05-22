# Description

This is a tool that takes a name of a Java Maven package or a POM file and returns the URLs of all its dependencies.

# Usage

There's a few flags available:
```
Usage of go-maven-resolver:
  -ignoreScopes string
    	Scopes to ignore. (default "provided,system,test")
  -reposFile string
    	Path file with repo URLs to check.
  -timeout int
    	HTTP request timeout in seconds. (default 2)
  -workers int
    	Number of fetching workers. (default 50)
```

But in the most basic terms the tool takes Maven formatted package names via `STDIN` and outputs URLs for all dependencies via `STDOUT`:
```
 $ echo commons-io:commons-io:2.4 | ./go-maven-resolver
https://repo.maven.apache.org/maven2/commons-io/commons-io/2.4/commons-io-2.4.pom
https://repo.maven.apache.org/maven2/org/apache/commons/commons-parent/25/commons-parent-25.pom
https://repo.maven.apache.org/maven2/org/apache/apache/9/apache-9.pom
```

# Reasoning

I've decided to write this because I could not find a way to achieve the same thing using Gradle or Maven.

For more details see [this Gradle Forum post](https://discuss.gradle.org/t/how-to-get-full-list-of-dependencies-and-their-meta/35825).

Using commands like `mvn dependency:list` or `mvn help:effective-pom` is too slow and give hard to parse output. It also attempts to download all of the necessary JARs and POMs which is unnecessary.
