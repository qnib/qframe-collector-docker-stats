package qframe_collector_docker_stats

import (
	"fmt"
	"log"
	"strings"
	"github.com/zpatrick/go-config"
	"github.com/fsouza/go-dockerclient"
	"github.com/qnib/qframe-types"
	"github.com/pkg/errors"
)

const (
	version = "0.0.3"
	pluginTyp = "collector"
)

// struct to keep info and channels to goroutine
// -> get heartbeats so that we know it's still alive
// -> allow for gracefully shutdown of the supervisor
type ContainerSupervisor struct {
	CntID 	string 			 // ContainerID
	CntName string			 // sanatized name of container
	Container docker.Container
	Com 	chan interface{} // Channel to communicate with goroutine
	cli 	*docker.Client
	qChan 	qtypes.QChan
}

func (cs ContainerSupervisor) Run() {
	log.Printf("[II] Start listener for: '%s' [%s]", cs.CntName, cs.CntID)
	// TODO: That is not realy straight forward...
	filter := map[string][]string{
		"id": []string{cs.CntID},
	}
	df := docker.ListContainersOptions{
		Filters: filter,
	}
	cnt, _ := cs.cli.ListContainers(df)
	if len(cnt) != 1 {
		log.Printf("[EE] Could not found excatly one container with id '%s'", cs.CntID)
		return
	}
	errChannel := make(chan error, 1)
	statsChannel := make(chan *docker.Stats)

	opts := docker.StatsOptions{
		ID:     cs.CntID,
		Stats:  statsChannel,
		Stream: true,
	}

	go func() {
		errChannel <- cs.cli.Stats(opts)
	}()

	for {
		select {
		case msg := <-cs.Com:
			switch msg {
			case "died":
				log.Printf("[DD] Container [%s]->'%s' died -> BYE!", cs.CntID, cs.CntName)
				return
			default:
				log.Printf("[DD] Container [%s]->'%s' got message from cs.Com: %v\n", cs.CntID, cs.CntName, msg)
			}
		case stats, ok := <-statsChannel:
			if !ok {
				err := errors.New(fmt.Sprintf("Bad response getting stats for container: %s", cs.CntID))
				log.Println(err.Error())
				return
			}
			qm := qtypes.NewQMsg("collector", "docker-stats")
			qm.Msg = "Send Metrics of "+cs.CntName
			qm.Data = qtypes.ContainerStats {
				Stats: stats,
				Container: cnt[0],
			}
			cs.qChan.Data.Send(qm)
		}
	}
}

type Plugin struct {
	qtypes.Plugin
	cli *docker.Client
	sMap map[string]ContainerSupervisor
}

func New(qChan qtypes.QChan, cfg config.Config, name string) (Plugin, error) {
	var err error
	p := Plugin{
		Plugin: qtypes.NewNamedPlugin(qChan, cfg, pluginTyp, name, version),
		sMap: map[string]ContainerSupervisor{},
	}
	return p, err
}

func (p *Plugin) Run() {
	var err error
	dockerHost := p.CfgStringOr("docker-host", "unix:///var/run/docker.sock")
	// Filter start/stop event of a container
	p.cli, err = docker.NewClient(dockerHost)
	if err != nil {
		p.Log("error", fmt.Sprintf("Could not connect fsouza/go-dockerclient to '%s': %v", dockerHost, err))
		return
	}
	info, err := p.cli.Info()
	if err != nil {
		p.Log("error", fmt.Sprintf("Error during Info(): %v >err> %s", info, err))
		return
	} else {
		p.Log("info", fmt.Sprintf("Connected to '%s' / v'%s' (SWARM: %s)", info.Name, info.ServerVersion, info.Swarm.LocalNodeState))
	}

	// List of current containers
	p.Log("info", fmt.Sprintf("Currently running containers: %d", info.ContainersRunning))
	bc := p.QChan.Back.Join()
	for {
		select {
		case msg := <-bc.Read:
			switch msg.(type) {
			case qtypes.QMsg:
				qm := msg.(qtypes.QMsg)
				switch qm.Data.(type){
				case qtypes.ContainerEvent:
					ce := qm.Data.(qtypes.ContainerEvent)
					// TODO: exec_* also includes the command - needs startswith()
					if ce.Event.Type == "container" && (strings.HasPrefix(ce.Event.Action, "exec_create") || strings.HasPrefix(ce.Event.Action, "exec_start")) {
						continue
					}
					p.Log("info", fmt.Sprintf("Received qtypes.ContainerEvent from back-channel: %s.%s", ce.Event.Type, ce.Event.Action))
					switch ce.Event.Type {
					case "container":
						switch ce.Event.Action {
						case "start":
							p.StartSupervisor(qm)
						case "die":
							p.sMap[ce.Event.Actor.ID].Com <- ce.Event.Action
						}
					}
				}
			}
		}
	}
}

func (p *Plugin) StartSupervisor(qm qtypes.QMsg) {
	ce := qm.Data.(qtypes.ContainerEvent)

	//p.Log("info", fmt.Sprintf("Let's go: container %s started!", event.Actor.Attributes["name"]))
	s := ContainerSupervisor{
		CntID: ce.Event.Actor.ID,
		CntName: ce.Event.Actor.Attributes["name"],
		Com: make(chan interface{}),
		cli: p.cli,
		qChan: p.QChan,
	}
	p.sMap[ce.Event.Actor.ID] = s
	go s.Run()
}
