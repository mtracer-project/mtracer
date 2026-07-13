package dockersetupcommand

import (
	"fmt"
	"strings"

	"github.com/mtracer-project/mtracer/parser"
)

type CustomQdiscContainerCommand struct {
	throttler         ContainerThrottler
	qdiscCmd          string
	targetContainerId string
	netInterface      string
}

func (s *CustomQdiscContainerCommand) Execute() error {
	// Build the helper container that shares the network namespace of the target container
	err := s.throttler.Throttle(s.qdiscCmd, s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to throttle container: %w", err)
	}

	return nil
}

func (s *CustomQdiscContainerCommand) Cleanup() error {
	err := s.throttler.Unthrottle(s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to unthrottle container: %w", err)
	}

	return nil
}

func NewCustomQdiscContainerCommand(cmd *parser.SetupCommandDTO, throttler ContainerThrottler) (*CustomQdiscContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	targetContainerId, ok := cmd.Args["containerId"].(string)
	if !ok || targetContainerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	qdiscCmd, ok := cmd.Args["qdiscCmd"].(string)
	if !ok || qdiscCmd == "" {
		return nil, fmt.Errorf("qdiscCmd argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	qdiscCmd = strings.TrimSpace(qdiscCmd)

	netInterface, ok := cmd.Args["netInterface"].(string)
	if !ok || netInterface == "" {
		netInterface = DEFAULT_NET_INTERFACE // default to eth0 if not specified
	}

	return &CustomQdiscContainerCommand{
		throttler:         throttler,
		qdiscCmd:          qdiscCmd,
		targetContainerId: targetContainerId,
		netInterface:      netInterface,
	}, nil
}
