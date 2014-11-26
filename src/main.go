package main

import (
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"strings"

	kani "./kanikousen"
	_ "./kanikousen/broaker"
	_ "./kanikousen/logger"
)

var opts struct {
	Verbose        bool   `short:"v" long:"verbose" description:"Enable verbose output"`
	DockerDaemon   string `short:"d" long:"dockerd" value-name:"ENDPOINT" default:"unix:///var/run/docker.sock" description:"Endpoint for docker daemon"`
	DockerRegistry string `short:"r" long:"registry" value-name:"URL" required:"true" description:"Url for docker registry used to fetch worker images"`
	Broaker        string `short:"b" long:"broaker" value-name:"BROAKER_CONFIG" required:"true" description:"Broaker configuration"`
	LogFile        string `short:"o" long:"logfile" value-name:"PATH" description:"File for logs output"`
	MaxWorkers     uint   `short:"n" long:"max-workers" value-name:"NUM" default:"5" description:"Number of jobs that would be run at the same time"`
	Queues         string `short:"t" long:"tasks" value-name:"ID1[,ID2...]" description:"List of target task ids separated by comma"`
	JobsCount      uint   `short:"c" long:"job-count" value-name:"NUM" description:"Wait until the number of jobs reached to this"`
	MaxFailCount   uint   `long:"failcount" value-name:"NUM" default:"5" description:"The number of fail count to be allowed for retrying"`
	Logger         string `short:"l" long:"logger" value-name:"LOGGER_CONFIG" default:"stdout:" description:"Logger configuration"`
}

func parseQueues(taskIdList string) []string {
	queues := strings.Split(taskIdList, ",")
	for i := 0; i < len(queues); i++ {
		if queues[i] == "" {
			queues = append(queues[0:i], queues[i+1:]...)
		}
	}
	return queues
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	loggerFactory, err := kani.LoggerFactoryFromSpec(opts.Logger)
	if err != nil {
		log.Fatalf("%s", err)
	}

	boat := &kani.Boat{
		MaxWorkers:     opts.MaxWorkers,
		LogWriter:      os.Stderr,
		DockerRegistry: opts.DockerRegistry,
		DockerDaemon:   opts.DockerDaemon,
		MaxFailCount:   opts.MaxFailCount,
		Schedule: &kani.Schedule{
			Queues:    parseQueues(opts.Queues),
			JobsCount: opts.JobsCount,
		},
		LoggerFactory: loggerFactory,
	}

	if opts.LogFile != "" {
		fp, err := os.OpenFile(opts.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Can't open '%s' for log output: %s\n",
				opts.LogFile, err)
		}
		defer fp.Close()
		log.SetOutput(fp)
		boat.LogWriter = fp
	}

	brk, err := kani.BroakerFromSpec(boat, opts.Broaker)
	if err != nil {
		log.Fatalln("Failed to create broaker instance: ", err)
	}
	defer brk.Close()

	boat.ServeLoop(brk)
}
