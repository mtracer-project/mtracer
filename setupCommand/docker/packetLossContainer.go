package dockersetupcommand

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mtracer-project/mtracer/parser"
)

const (
	PACKET_LOSS_CONTAINER_CMD = "netem loss %f%%"
)

type PacketLossContainerCommand struct {
	throttler         ContainerThrottler
	loss              float64
	targetContainerId string
	netInterface      string
}

func (s *PacketLossContainerCommand) Execute() error {
	// Build the helper container that shares the network namespace of the target container
	lossContainerCmd := fmt.Sprintf(PACKET_LOSS_CONTAINER_CMD, s.loss)
	err := s.throttler.Throttle(lossContainerCmd, s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to throttle container: %w", err)
	}

	return nil
}

func (s *PacketLossContainerCommand) Cleanup() error {
	err := s.throttler.Unthrottle(s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to unthrottle container: %w", err)
	}

	return nil
}

func NewPacketLossContainerCommand(cmd *parser.SetupCommandDTO, throttler ContainerThrottler) (*PacketLossContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	targetContainerId, ok := cmd.Args["containerId"].(string)
	if !ok || targetContainerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	lossStr, ok := cmd.Args["loss"].(string)
	if !ok || lossStr == "" {
		return nil, fmt.Errorf("loss argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	lossStr = strings.TrimSpace(lossStr)
	lossStr = strings.TrimSuffix(lossStr, "%") // Remove % if present

	loss, err := strconv.ParseFloat(lossStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid loss format for setup command '%s': %w", cmd.Cmd, err)
	}

	netInterface, ok := cmd.Args["netInterface"].(string)
	if !ok || netInterface == "" {
		netInterface = DEFAULT_NET_INTERFACE // default to eth0 if not specified
	}

	return &PacketLossContainerCommand{
		throttler:         throttler,
		loss:              loss,
		targetContainerId: targetContainerId,
		netInterface:      netInterface,
	}, nil
}
