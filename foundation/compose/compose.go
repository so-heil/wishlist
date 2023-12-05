// Package compose uses docker-compose style descriptions to create and manage test container
package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// command is a helper function that runs an executable
type command func(arg ...string) ([]byte, error)

type cleaner func() error

type Compose struct {
	cleanUpFuncs []cleaner
	cmd          command
}

func New(content string) (*Compose, error) {
	f, err := os.CreateTemp("", "compose-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp file for compose: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return nil, fmt.Errorf("write content on temp file: %w", err)
	}
	name := f.Name()

	dockerCompose := func(arg ...string) ([]byte, error) {
		return run("docker-compose", []string{"-f", name}, arg)
	}
	compose := &Compose{
		cmd: dockerCompose,
	}

	// compose-compose config does some sort of format-checking for the compose file
	_, err = compose.cmd("config")
	if err != nil {
		return nil, err
	}

	compose.addCleaner(func() error {
		return os.Remove(name)
	})

	return compose, err
}

func (compose *Compose) Down() error {
	if _, err := compose.cmd("down"); err != nil {
		return err
	}
	return nil
}

func (compose *Compose) Up() (map[string]Container, error) {
	if _, err := compose.cmd("up", "-d"); err != nil {
		return nil, err
	}
	compose.addCleaner(compose.Down)

	return compose.Containers()
}

func (compose *Compose) Close() error {
	for i := len(compose.cleanUpFuncs) - 1; i >= 0; i-- {
		f := compose.cleanUpFuncs[i]
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (compose *Compose) Containers() (map[string]Container, error) {
	jsn, err := compose.cmd("ps", "--format", "json")
	if err != nil {
		return nil, err
	}

	var ctrs []composeContainer
	if err := json.Unmarshal(jsn, &ctrs); err != nil {
		return nil, fmt.Errorf("json unmarshal compose ps: %w", err)
	}

	res := make(map[string]Container)
	for _, cc := range ctrs {
		res[cc.Service] = Container{
			ID:    cc.ID,
			Host:  cc.host(),
			Image: cc.Image,
		}
	}
	return res, nil
}

func (compose *Compose) addCleaner(f cleaner) {
	if f != nil {
		compose.cleanUpFuncs = append(compose.cleanUpFuncs, f)
	}
}

func run(name string, initArg, arg []string) ([]byte, error) {
	cmd := exec.Command(name, initArg...)
	cmd.Args = append(cmd.Args, arg...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run command:\ncommand: %s\nerror: %w\noutput: %s", cmd.String(), err, string(out))
	}

	return out, nil
}
