package compose

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

const waitInterval = 500

type Container struct {
	ID    string
	Host  string
	Image string
}

type composeContainer struct {
	ID         string `json:"ID"`
	Image      string `json:"Image"`
	Service    string `json:"Service"`
	Publishers []struct {
		URL           string `json:"URL"`
		PublishedPort int    `json:"PublishedPort"`
	} `json:"Publishers"`
}

func (cc *composeContainer) host() string {
	var h string
	for _, publisher := range cc.Publishers {
		if publisher.URL != "" && publisher.PublishedPort != 0 {
			h = net.JoinHostPort(publisher.URL, strconv.Itoa(publisher.PublishedPort))
			break
		}
	}
	return h
}

func (container *Container) DumpLogs() error {
	out, err := docker("logs", container.ID)
	if err != nil {
		return err
	}
	fmt.Printf("contaier logs:\n%s", string(out))
	return nil
}

func (container *Container) WaitForHealthy(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := docker("inspect", container.ID, "--format", "{{.State.Health.Status}}")
		if err != nil {
			return err
		}

		if string(out) == "healthy\n" {
			return nil
		}
		time.Sleep(waitInterval * time.Millisecond)
	}
}

func docker(arg ...string) ([]byte, error) {
	return run("docker", nil, arg)
}
