package fetcher

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

/* List of Maven repo URLs to try when searching for POMs */
var DefaultRepos = []string{
	"https://repo.maven.apache.org/maven2",
	"https://dl.google.com/dl/android/maven2",
	"https://repository.sonatype.org/content/groups/sonatype-public-grid",
}

type Result struct {
	Url  string
	Repo string
	Data io.ReadCloser
}

type Job struct {
	result chan *Result
	path   string
	repo   string
}

func NewJob(result chan *Result, path, repo string) *Job {
	return &Job{
		result: result,
		path:   path,
		repo:   repo,
	}
}

/* In order to avoid hitting the 'socket: too many open files' error
 * We manage a pool of workers that do the HTTP requests to Maven repos. */
type Fetcher struct {
	limit   int         /* max number of workers in pool */
	timeout int         /* http request timeout in seconds */
	retries int         /* number of retries on non-404 error */
	Queue   chan *Job   /* channel for queuing jobs */
	repos   []string    /* list of Repo URLs to try */
	l       *log.Logger /* for logging to stderr */
}

func (r *Result) String() string {
	return fmt.Sprintf("<Result Url=%s >", r.Url)
}

func New(retries, limit, timeout int, repos []string, l *log.Logger) Fetcher {
	f := Fetcher{
		retries: retries,
		limit:   limit,
		timeout: timeout,
		Queue:   make(chan *Job, limit),
		repos:   repos,
		l:       l,
	}
	/* start workers */
	for i := 0; i < f.limit; i++ {
		go f.Worker()
	}
	return f
}

func (p *Fetcher) retryFetch(url string) (io.ReadCloser, error) {
	var resp http.Response
	for r := 1; r <= p.retries; r++ {
		client := &http.Client{
			Timeout: time.Duration(p.timeout) * time.Second,
		}
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}
		/* 404 must mean it doesn't exist. */
		if resp.StatusCode == http.StatusNotFound {
			break
		}
		/* Any other code might be a transient issue. */
		p.l.Printf("warning: retrying %d/%d due to status: %d, url: %s",
			r, p.retries, resp.StatusCode, url)
	}
	return nil, fmt.Errorf("failed to fetch with: %d", resp.StatusCode)
}

func (p *Fetcher) tryRepo(repo, path string) (*Result, error) {
	var err error
	url := repo + "/" + path
	data, err := p.retryFetch(url)
	if err != nil {
		return nil, fmt.Errorf("error: '%s' for: %s", err, url)
	}
	return &Result{url, repo, data}, nil
}

func (p *Fetcher) tryRepos(job *Job, repos []string) {
	for _, repo := range repos {
		rval, err := p.tryRepo(repo, job.path)
		if err == nil {
			job.result <- rval
			return
		}
	}

	job.result <- &Result{}
}

func (p *Fetcher) Worker() {
	for job := range p.Queue {
		var repos []string = p.repos

		/* Repo can be provided in the job */
		if job.repo != "" {
			repos = []string{job.repo}
		}

		p.tryRepos(job, repos)
	}
}
