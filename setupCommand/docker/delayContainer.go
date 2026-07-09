package dockersetupcommand

import (
	"fmt"
	"time"

	"github.com/mtrace-project/mtrace/parser"
)

const (
	DELAY_CONTAINER_CMD = "netem delay %dms"
)

type DelayContainerCommand struct {
	throttler         ContainerThrottler
	delay             time.Duration
	targetContainerId string
	netInterface      string
}

func (s *DelayContainerCommand) Execute() error {
	// Build the helper container that shares the network namespace of the target container
	delayContainerCmd := fmt.Sprintf(DELAY_CONTAINER_CMD, s.delay.Milliseconds())
	err := s.throttler.Throttle(delayContainerCmd, s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to throttle container: %w", err)
	}

	return nil
}

func (s *DelayContainerCommand) Cleanup() error {
	err := s.throttler.Unthrottle(s.netInterface, s.targetContainerId)
	if err != nil {
		return fmt.Errorf("failed to unthrottle container: %w", err)
	}

	return nil
}

func NewDelayContainerCommand(cmd *parser.SetupCommandDTO, throttler ContainerThrottler) (*DelayContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	targetContainerId, ok := cmd.Args["containerId"].(string)
	if !ok || targetContainerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	delayStr, ok := cmd.Args["delay"].(string)
	if !ok || delayStr == "" {
		return nil, fmt.Errorf("delay argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid delay format for setup command '%s': %w", cmd.Cmd, err)
	}

	netInterface, ok := cmd.Args["netInterface"].(string)
	if !ok || netInterface == "" {
		netInterface = DEFAULT_NET_INTERFACE // default to eth0 if not specified
	}

	return &DelayContainerCommand{
		throttler:         throttler,
		delay:             delay,
		targetContainerId: targetContainerId,
		netInterface:      netInterface,
	}, nil
}
