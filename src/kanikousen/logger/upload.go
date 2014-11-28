package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"

	k "../"
)

type UploadLogger struct {
	job        *k.Job
	paramName  string
	stdoutTmpl string
	stderrTmpl string
}

func init() {
	k.RegisterLogger("upload", NewUploadLogger)
}

func NewUploadLogger(job *k.Job, opts map[string]string) (k.Logger, error) {
	paramName, ok := opts["param_name"]
	if !ok {
		return nil, fmt.Errorf("'param_name' required")
	}
	stdoutTmpl, ok := opts["stdout_tmpl"]
	if !ok {
		return nil, fmt.Errorf("'stdout_tmpl' required")
	}
	stderrTmpl, ok := opts["stderr_tmpl"]
	if !ok {
		return nil, fmt.Errorf("'stderr_tmpl' required")
	}

	logger := &UploadLogger{
		job:        job,
		paramName:  paramName,
		stdoutTmpl: stdoutTmpl,
		stderrTmpl: stderrTmpl,
	}
	return logger, nil
}

func (logger *UploadLogger) deliver(in io.Reader, url string) {
	pipeR, pipeW := io.Pipe()
	writer := multipart.NewWriter(pipeW)

	go func() {
		defer pipeW.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile(logger.paramName, filepath.Base(url))
		if err != nil {
			log.Println("failed to create multipart message:", err)
			return
		}
		_, err = io.Copy(part, in)
		if err != nil {
			log.Println("failed to write to multipart stream:", err)
		}
	}()

	resp, err := http.Post(url, writer.FormDataContentType(), pipeR)
	if err != nil {
		log.Printf("failed to post log to %s: %s", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf, _ := ioutil.ReadAll(resp.Body)
		log.Printf("failed to post log to %s: status=%s, body=%s", url, resp.Status, string(buf))
	}
}

func (logger *UploadLogger) Yield(stdout, stderr io.Reader) {
	outGuard := make(chan bool)
	errGuard := make(chan bool)

	go func() {
		logger.deliver(stdout, fmt.Sprintf(logger.stdoutTmpl, logger.job.Id))
		outGuard <- true
	}()
	go func() {
		logger.deliver(stderr, fmt.Sprintf(logger.stderrTmpl, logger.job.Id))
		errGuard <- true
	}()

	<-outGuard
	<-errGuard
}
