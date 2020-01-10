package provisioning

import (
	"os/exec"
	"strings"
)

// LinuxRunner is a ProvisioningRunner that runs on the executing host only.
// The Nomad configurations used with this runner will need to avoid port
// conflicts!
type LinuxRunner struct{}

func (runner *LinuxRunner) Open() error {}

func (runner *LinuxRunner) Run(script string) error {
	commands := strings.Split(script, "\n")
	for _, command := range commands {
		cmd := exec.Command(strings.TrimSpace(command))
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (runner *LinuxRunner) Copy(local, remote string) error {
	// TODO(tgross): yeah, yeah, this is hacky as heck
	cmd := exec.Command("cp", "-rf", local, remote)
	return cmd.Run()
}

func (runner *LinuxRunner) Close() {}
