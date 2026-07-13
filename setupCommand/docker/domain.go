package dockersetupcommand

import (
	"context"
	"fmt"
	"strings"

	"github.com/mtracer-project/mtracer/parser"

	"github.com/moby/moby/client"
)

type DockerSetupCommand interface {
	Execute() error
	Cleanup() error
}

func NewDockerSetupCommand(cmd *parser.SetupCommandDTO, handler *DockerHandler) (DockerSetupCommand, error) {
	throttler := NewDockerContainerThrottler(handler, handler, handler, handler)

	switch strings.ToLower(cmd.Cmd) {
	case "killcontainer":
		return NewKillContainerCommand(cmd, handler, handler)
	case "stopcontainer":
		return NewStopContainerCommand(cmd, handler, handler)
	case "startcontainer":
		return NewStartContainerCommand(cmd, handler, handler)
	case "pausecontainer":
		return NewPauseContainerCommand(cmd, handler, handler)
	case "unpausecontainer":
		return NewUnpauseContainerCommand(cmd, handler, handler)
	case "execcontainer":
		return NewExecContainerCommand(cmd, handler)
	case "delaycontainer":
		return NewDelayContainerCommand(cmd, throttler)
	case "packetlosscontainer":
		return NewPacketLossContainerCommand(cmd, throttler)
	case "customqdisccontainer":
		return NewCustomQdiscContainerCommand(cmd, throttler)
	case "composeup":
		return NewComposeUpCommand(cmd, handler.baseDir, handler.ctx)
	default:
		return nil, fmt.Errorf("unsupported docker setup command type: %s", cmd.Type)
	}
}

type ContainerCommandExecutor interface {
	Execute(containerId string, cmd string) error
}

type HelperContainerBuilder interface {
	Build(targetContainerId string, netInterface string, cmd string) (string, error)
}

type ContainerStarter interface {
	Start(containerId string) error
}

type ContainerStopper interface {
	Stop(containerId string) error
}

type ContainerPauser interface {
	Pause(containerId string) error
}

type ContainerUnpauser interface {
	Unpause(containerId string) error
}

type ContainerKiller interface {
	Kill(containerId string) error
}

var (
	_ ContainerCommandExecutor = (*DockerHandler)(nil)
	_ HelperContainerBuilder   = (*DockerHandler)(nil)
	_ ContainerStarter         = (*DockerHandler)(nil)
	_ ContainerStopper         = (*DockerHandler)(nil)
	_ ContainerPauser          = (*DockerHandler)(nil)
	_ ContainerUnpauser        = (*DockerHandler)(nil)
	_ ContainerKiller          = (*DockerHandler)(nil)
)

type DockerHandler struct {
	client  *client.Client
	baseDir string
	ctx     context.Context
}

func NewDockerHandler(cli *client.Client, baseDir string, ctx context.Context) *DockerHandler {
	return &DockerHandler{
		client:  cli,
		baseDir: baseDir,
		ctx:     ctx,
	}
}
