package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func defaultRepos() []string {
	return []string{
		"https://repo.maven.apache.org/maven2",
		"https://dl.google.com/dl/android/maven2",
		"https://repository.sonatype.org/content/groups/sonatype-public-grid",
	}
}

type FetcherResult struct {
	url  string
	repo string
	data io.ReadCloser
}

type FetcherJob struct {
	result chan *FetcherResult
	path   string
	repo   string
}

/* In order to avoid hitting the 'socket: too many open files' error
 * We manage a pool of workers that do the HTTP requests to Maven repos. */
type FetcherPool struct {
	limit   int              /* max number of workers in pool */
	timeout int              /* http request timeout in seconds */
	queue   chan *FetcherJob /* channel for queuing jobs */
	repos   []string         /* list of repo URLs to try */
}

func (r *FetcherResult) String() string {
	return fmt.Sprintf("<FetcherResult url=%s >", r.url)
}

func NewFetcherPool(limit, timeout int, repos []string) FetcherPool {
	f := FetcherPool{
		limit:   limit,
		timeout: timeout,
		queue:   make(chan *FetcherJob, limit),
		repos:   repos,
	}
	/* start workers */
	for i := 0; i < f.limit; i++ {
		go f.Worker()
	}
	return f
}

func (p *FetcherPool) Fetch(url string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Duration(p.timeout) * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch with: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (p *FetcherPool) TryRepo(repo, path string) (*FetcherResult, error) {
	var err error
	url := repo + "/" + path
	data, err := p.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("error: '%s' for: %s", err, url)
	}
	return &FetcherResult{url, repo, data}, nil
}

func (p *FetcherPool) TryRepos(job *FetcherJob, repos []string) {
	for _, repo := range repos {
		rval, err := p.TryRepo(repo, job.path)
		if err == nil {
			job.result <- rval
			return
		}
	}

	job.result <- &FetcherResult{}
}

func (p *FetcherPool) Worker() {
	for job := range p.queue {
		var repos []string = p.repos

		/* repo can be provided in the job */
		if job.repo != "" {
			repos = []string{job.repo}
		}

		p.TryRepos(job, repos)
	}
}
