package main

import (
	"log"
	"fmt"

	"github.com/zpatrick/go-config"

	"github.com/qnib/qframe-types"
	"github.com/qnib/qframe-collector-docker-stats/lib"
	"github.com/qnib/qframe-collector-docker-events/lib"
)

func Run(qChan qtypes.QChan, cfg *config.Config, name string) {
	p, _ := qframe_collector_docker_stats.New(qChan, cfg, name)
	p.Run()
}

func main() {
	qChan := qtypes.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"collector.events.docker-host": "unix:///var/run/docker.sock",
		"filter.id.send-back": "docker-event",
		"filter.id.inputs": "events",
	}

	cfg := config.NewConfig(
		[]config.Provider{
			config.NewStatic(cfgMap),
		},
	)
	// start docker-events
	pe, err := qframe_collector_docker_events.New(qChan, cfg, "events")
	go pe.Run()
	// start docker-stats
	p, err := qframe_collector_docker_stats.New(qChan, cfg, "stats")
	if err != nil {
		log.Printf("[EE] Failed to create collector: %v", err)
		return
	}
	go p.Run()
	dc := qChan.Data.Join()
	bc := qChan.Back.Join()
	done := false
	for {
		select {
		case msg := <-dc.Read:
			switch msg.(type) {
				case qtypes.ContainerStats:
					qs := msg.(qtypes.ContainerStats)
					if qs.IsLastSource("docker-stats") {
						fmt.Printf("%v\n", qs.Stats)
						done = true
					}
			}
		case msg := <-bc.Read:
			qm := msg.(qtypes.QMsg)
			fmt.Printf("BACK # Received [%s]: %s\n", qm.Source, qm.Msg)
		}
		if done {
			break
		}
	}
}
