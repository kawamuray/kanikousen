package logger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	k "../"
)

type StdoutLogger struct {
	job *k.Job
}

func init() {
	k.RegisterLogger("stdout", NewStdoutLogger)
}

func NewStdoutLogger(job *k.Job, _ map[string]string) (k.Logger, error) {
	logger := &StdoutLogger{job: job}
	return logger, nil
}

func (logger *StdoutLogger) writeWithPrefix(in io.Reader, out io.Writer) {
	prefix := fmt.Sprintf("job-%d: ", logger.job.Id)

	buf_in := bufio.NewReader(in)
	wasPrefix := false
	for {
		line, ispfx, err := buf_in.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Printf("error while reading log from job %d: %s\n", logger.job.Id, err)
				continue
			}
		}
		if !wasPrefix {
			fmt.Fprint(out, prefix)
		}
		out.Write(line)
		if !ispfx {
			fmt.Fprint(out, "\n")
		}
		wasPrefix = ispfx
	}
}

func (logger *StdoutLogger) Yield(stdout, stderr io.Reader) {
	ch_out := make(chan bool)
	ch_err := make(chan bool)

	go func() {
		logger.writeWithPrefix(stdout, os.Stdout)
		ch_out <- true
	}()
	go func() {
		logger.writeWithPrefix(stderr, os.Stderr)
		ch_err <- true
	}()

	<-ch_out
	<-ch_err
}
