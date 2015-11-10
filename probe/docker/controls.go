package docker

import (
	"log"

	docker_client "github.com/fsouza/go-dockerclient"

	"github.com/weaveworks/scope/probe/controls"
	"github.com/weaveworks/scope/report"
	"github.com/weaveworks/scope/xfer"
)

// Control IDs used by the docker intergation.
const (
	StopContainer    = "docker_stop_container"
	StartContainer   = "docker_start_container"
	RestartContainer = "docker_restart_container"
	PauseContainer   = "docker_pause_container"
	UnpauseContainer = "docker_unpause_container"
	AttachContainer  = "docker_attach_container"

	waitTime = 10
)

func (r *registry) stopContainer(containerID string, _ xfer.Request) xfer.Response {
	log.Printf("Stopping container %s", containerID)
	return xfer.ResponseError(r.client.StopContainer(containerID, waitTime))
}

func (r *registry) startContainer(containerID string, _ xfer.Request) xfer.Response {
	log.Printf("Starting container %s", containerID)
	return xfer.ResponseError(r.client.StartContainer(containerID, nil))
}

func (r *registry) restartContainer(containerID string, _ xfer.Request) xfer.Response {
	log.Printf("Restarting container %s", containerID)
	return xfer.ResponseError(r.client.RestartContainer(containerID, waitTime))
}

func (r *registry) pauseContainer(containerID string, _ xfer.Request) xfer.Response {
	log.Printf("Pausing container %s", containerID)
	return xfer.ResponseError(r.client.PauseContainer(containerID))
}

func (r *registry) unpauseContainer(containerID string, _ xfer.Request) xfer.Response {
	log.Printf("Unpausing container %s", containerID)
	return xfer.ResponseError(r.client.UnpauseContainer(containerID))
}

func (r *registry) attachContainer(containerID string, req xfer.Request) xfer.Response {
	pipe, err := controls.Pipes.NewPipe(req.AppID)
	if err != nil {
		xfer.ResponseError(err)
	}
	err = r.client.AttachToContainer(docker_client.AttachToContainerOptions{
		Container:    containerID,
		RawTerminal:  true,
		InputStream:  pipe,
		OutputStream: pipe,
		ErrorStream:  pipe,
	})
	if err != nil {
		xfer.ResponseError(err)
	}
	return xfer.Response{
		Pipe: pipe.ID(),
	}
}

func captureContainerID(f func(string, xfer.Request) xfer.Response) func(xfer.Request) xfer.Response {
	return func(req xfer.Request) xfer.Response {
		_, containerID, ok := report.ParseContainerNodeID(req.NodeID)
		if !ok {
			return xfer.ResponseErrorf("Invalid ID: %s", req.NodeID)
		}
		return f(containerID, req)
	}
}

func (r *registry) registerControls() {
	controls.Register(StopContainer, captureContainerID(r.stopContainer))
	controls.Register(StartContainer, captureContainerID(r.startContainer))
	controls.Register(RestartContainer, captureContainerID(r.restartContainer))
	controls.Register(PauseContainer, captureContainerID(r.pauseContainer))
	controls.Register(UnpauseContainer, captureContainerID(r.unpauseContainer))
	controls.Register(AttachContainer, captureContainerID(r.attachContainer))
}
