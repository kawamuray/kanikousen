package kanikousen

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

type RepoCache struct {
	boat  *Boat
	cache map[string]string
	lock  *sync.Mutex
}

func NewRepoCache(boat *Boat) *RepoCache {
	return &RepoCache{
		boat:  boat,
		cache: make(map[string]string),
		lock:  new(sync.Mutex),
	}
}

func (rc *RepoCache) getImageIdForTag(tag string) (string, error) {
	// Split registry prefix and repository
	parts := strings.SplitN(tag, "/", 2)
	resp, err := http.Get(fmt.Sprintf(
		"http://%s/v1/repositories/%s/tags/latest",
		rc.boat.DockerRegistry, parts[1]))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Discard heading/trailing quotes `"`
	id := string(body)
	return id[:len(id)-1][1:], nil
}

func (rc *RepoCache) refreshCache(dc *docker.Client) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	imgs, err := dc.ListImages(false)
	if err != nil {
		return err
	}
	rc.cache = make(map[string]string)
	for _, img := range imgs {
		for _, repo := range img.RepoTags {
			rc.cache[repo] = img.ID
		}
	}
	return nil
}

func (rc *RepoCache) pullImage(dc *docker.Client, repo string) error {
	opts := docker.PullImageOptions{
		Repository: repo,
		// parameter "registry" seems completely meaningless at least version 1.2.0
		// Registry:      rc.boat.DockerRegistry,
		Tag:           "latest",
		OutputStream:  rc.boat.LogWriter,
		RawJSONStream: false,
	}
	err := dc.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		return err
	}
	return rc.refreshCache(dc)
}

func (rc *RepoCache) prepareImage(dc *docker.Client, repo string, ch chan error) {
	repoFull := repo + ":latest"

	cachedId, ok := rc.cache[repoFull]
	if ok {
		imageId, err := rc.getImageIdForTag(repo)
		if err != nil {
			// api call failed, but still able to continue with assuming cache expired
			log.Println("warning: failed to get image id from registry: ", err)
		} else if imageId == cachedId {
			ch <- nil
			return
		}
	}

	err := rc.pullImage(dc, repo)
	if err != nil {
		ch <- nil
		return
	}

	_, ok = rc.cache[repoFull]
	if ok {
		ch <- nil
		return
	}

	ch <- fmt.Errorf("No such image available: %s", repoFull)
}

func (rc *RepoCache) Request(dc *docker.Client, repo string) chan error {
	ch := make(chan error)
	go rc.prepareImage(dc, repo, ch)
	return ch
}
