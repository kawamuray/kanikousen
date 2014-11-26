package kanikousen

import (
	"github.com/fsouza/go-dockerclient"
	"io"
	"log"
	"reflect"
)

type Boat struct {
	MaxWorkers     uint
	LogWriter      io.Writer
	DockerRegistry string
	DockerDaemon   string
	MaxFailCount   uint
	Schedule       *Schedule
	LoggerFactory  func(*Job) (Logger, error)
}

type execResult struct {
	Job    *Job
	Status int
	Err    error
}

func doWork(spwn *Spawner, chanJobs chan *Job, chanStatus chan execResult) {
	for job := range chanJobs {
		status, err := spwn.RunJob(job)
		chanStatus <- execResult{
			Job:    job,
			Status: status,
			Err:    err,
		}
	}
}

func (boat *Boat) startWorkers(chanStatus chan execResult) (chan *Job, error) {
	chanJobs := make(chan *Job)
	repoCache := NewRepoCache(boat)

	for i := uint(0); i < boat.MaxWorkers; i++ {
		spwn, err := NewSpawner(boat, repoCache, boat.DockerDaemon)
		if err != nil {
			return nil, err
		}
		go doWork(spwn, chanJobs, chanStatus)
	}

	return chanJobs, nil
}

func isUserFault(err error) bool {
	return reflect.TypeOf(err) == reflect.TypeOf(new(docker.Error)) ||
		err == docker.ErrNoSuchImage ||
		err == docker.ErrMissingRepo
}

func (boat *Boat) ServeLoop(brk Broaker) {
	chanStatus := make(chan execResult)
	chanTasks, err := boat.startWorkers(chanStatus)
	if err != nil {
		panic(err)
	}

	go brk.FetchLoop(chanTasks, boat.Schedule)

	for r := range chanStatus {
		if r.Err != nil {
			log.Printf("ERROR: job %d(task: %s) failed by error: %s\n",
				r.Job.Id, r.Job.TaskId, r.Err)
			if isUserFault(r.Err) {
				// Oops, this must be an error by user mistake,
				// won't be recovered by retry.
				brk.Failed(r.Job, r.Err)
				continue
			}
		} else if r.Status != 0 {
			log.Printf("ERROR: job %d(task: %s) exit with %d\n",
				r.Job.Id, r.Job.TaskId, r.Status)
		}
		brk.Finish(r.Job, r.Status)
	}
}
