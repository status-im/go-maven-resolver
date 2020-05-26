package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	Result chan *Result
	Path   string
	Repo   string
}

/* In order to avoid hitting the 'socket: too many open files' error
 * We manage a pool of workers that do the HTTP requests to Maven repos. */
type Pool struct {
	limit   int       /* max number of workers in pool */
	timeout int       /* http request timeout in seconds */
	Queue   chan *Job /* channel for queuing jobs */
	repos   []string  /* list of Repo URLs to try */
}

func (r *Result) String() string {
	return fmt.Sprintf("<Result Url=%s >", r.Url)
}

func NewPool(limit, timeout int, repos []string) Pool {
	f := Pool{
		limit:   limit,
		timeout: timeout,
		Queue:   make(chan *Job, limit),
		repos:   repos,
	}
	/* start workers */
	for i := 0; i < f.limit; i++ {
		go f.Worker()
	}
	return f
}

func (p *Pool) Fetch(url string) (io.ReadCloser, error) {
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

func (p *Pool) TryRepo(repo, path string) (*Result, error) {
	var err error
	url := repo + "/" + path
	data, err := p.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("error: '%s' for: %s", err, url)
	}
	return &Result{url, repo, data}, nil
}

func (p *Pool) TryRepos(job *Job, repos []string) {
	for _, repo := range repos {
		rval, err := p.TryRepo(repo, job.Path)
		if err == nil {
			job.Result <- rval
			return
		}
	}

	job.Result <- &Result{}
}

func (p *Pool) Worker() {
	for job := range p.Queue {
		var repos []string = p.repos

		/* Repo can be provided in the job */
		if job.Repo != "" {
			repos = []string{job.Repo}
		}

		p.TryRepos(job, repos)
	}
}
