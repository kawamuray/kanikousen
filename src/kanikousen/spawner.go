package kanikousen

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"io"
	"log"
)

type Spawner struct {
	dcl       *docker.Client
	boat      *Boat
	repoCache *RepoCache
}

func NewSpawner(boat *Boat, repoCache *RepoCache, endpoint string) (*Spawner, error) {
	dcl, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	spwn := &Spawner{
		dcl:       dcl,
		boat:      boat,
		repoCache: repoCache,
	}
	return spwn, nil
}

func (spwn *Spawner) dispatchLogs(ctId string, job *Job) {
	logger, err := spwn.boat.LoggerFactory(job)
	if err != nil {
		log.Printf("failed to create logger for job %d: %s\n", job.Id, err)
		return
	}

	out_r, out_w := io.Pipe()
	defer out_r.Close()
	err_r, err_w := io.Pipe()
	defer err_r.Close()

	guard := make(chan bool)
	go func() {
		logger.Yield(out_r, err_r)
		guard <- true
	}()

	err = spwn.dcl.Logs(docker.LogsOptions{
		Container:    ctId,
		Stdout:       true,
		Stderr:       true,
		OutputStream: out_w,
		ErrorStream:  err_w,
		Follow:       false,
		Timestamps:   false,
		RawTerminal:  false,
	})
	if err != nil {
		log.Printf("failed to retrieve container logs for job %d: %s\n", job.Id, err)
	}
	out_w.Close()
	err_w.Close()
	<-guard
}

func (spwn *Spawner) RunJob(job *Job) (int, error) {
	repo := spwn.boat.DockerRegistry + "/" + job.TaskId

	err := <-spwn.repoCache.Request(spwn.dcl, repo)
	if err != nil {
		return -1, err
	}

	ct, err := spwn.dcl.CreateContainer(docker.CreateContainerOptions{
		Name: fmt.Sprintf("job-%d", job.Id),
		Config: &docker.Config{
			Cmd:   job.Args,
			Image: repo,
		},
	})
	if err != nil {
		return -1, err
	}
	err = spwn.dcl.StartContainer(ct.ID, &docker.HostConfig{})
	if err != nil {
		return -1, err
	}

	status, err := spwn.dcl.WaitContainer(ct.ID)
	if err != nil {
		err = fmt.Errorf("error while waiting finish of job %d(task: %s): %s",
			job.Id, job.TaskId, err)
	}

	spwn.dispatchLogs(ct.ID, job)
	err = spwn.dcl.RemoveContainer(docker.RemoveContainerOptions{ID: ct.ID})

	return status, err
}
