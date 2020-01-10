package provisioning

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TODO(tgross): pass a logger rather than fmt.Println

// SSHRunner is a ProvisioningRunner that deploys via ssh.
// Terraform does all of this more elegantly and portably in its
// ssh communicator, but by shelling out we avoid pulling in TF's as
// a Nomad dependency, and avoid some long-standing issues with
// connections to Windows servers. The tradeoff is losing portability
// but in practice we're always going to run this from a Unixish
// machine.
type SSHRunner struct {
	Key  string // `json:"key"`
	User string // `json:"user"`
	Host string // `json:"host"`
	Port int    // `json:"port"`

	controlSockPath string
	ctx             context.Context
	cancelFunc      context.CancelFunc
	muxErr          chan error
}

// Open establishes the ssh connection. We keep this connection open
// so that we can multiplex subsequent ssh connections.
func (runner *SSHRunner) Open() error {
	fmt.Printf("[DEBUG] opening connection to %s\n", runner.Host)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	runner.ctx = ctx
	runner.cancelFunc = cancel
	runner.muxErr = make(chan error)

	home, _ := os.UserHomeDir()
	runner.controlSockPath = filepath.Join(
		home, ".ssh",
		fmt.Sprintf("ssh-control-%s-%d.sock", runner.Host, os.Getpid()))

	cmd := exec.CommandContext(ctx,
		"ssh",
		"-M", "-S", runner.controlSockPath,
		"-o", "StrictHostKeyChecking=no", // we're those terrible cloud devs
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", runner.Key,
		"-p", fmt.Sprintf("%v", runner.Port),
		fmt.Sprintf("%s@%s", runner.User, runner.Host),
	)

	go func() {
		// will block until command completes, we canel, or timeout
		runner.muxErr <- cmd.Run()
	}()
	return nil
}

func (runner *SSHRunner) Run(script string) error {
	commands := strings.Split(script, "\n")
	for _, command := range commands {
		err := runner.run(strings.TrimSpace(command))
		if err != nil {
			runner.cancelFunc()
			return err
		}
	}
	return nil
}

func (runner *SSHRunner) run(command string) error {
	if runner.controlSockPath == "" {
		return fmt.Errorf("Run failed: you need to call Open() first")
	}
	fmt.Printf("[DEBUG] running '%s' on %s\n", command, runner.Host)
	cmd := exec.CommandContext(runner.ctx,
		"ssh",
		"-S", runner.controlSockPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", runner.Key,
		"-p", fmt.Sprintf("%v", runner.Port),
		fmt.Sprintf("%s@%s", runner.User, runner.Host),
		command)

	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	if err != nil && err != context.Canceled {
		runner.cancelFunc()
		return err
	}
	return nil
}

// Copy uploads the local path to the remote path.
// TODO: would be nice to set file owner/mode here
func (runner *SSHRunner) Copy(local, remote string) error {
	fmt.Printf("[DEBUG] copying '%s' to '%s' on %s\n", local, remote, runner.Host)
	remoteDir, remoteFileName := filepath.Split(remote)

	// we stage to /tmp so that we can handle root-owned files
	tempPath := fmt.Sprintf("/tmp/%s", remoteFileName)

	cmd := exec.CommandContext(runner.ctx,
		"scp", "-r",
		"-o", fmt.Sprintf("ControlPath=%s", runner.controlSockPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", runner.Key,
		"-P", fmt.Sprintf("%v", runner.Port),
		local,
		fmt.Sprintf("%s@%s:%s", runner.User, runner.Host, tempPath))

	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil && err != context.Canceled {
		fmt.Printf("%s\n", stdoutStderr)
		runner.cancelFunc()
		return err
	}

	if isDir(local) {
		// this is a little inefficient but it lets us merge the contents of
		// a bundled directory with existing directories
		err = runner.Run(
			fmt.Sprintf("sudo mkdir -p %s; sudo cp -R %s %s; sudo rm -r %s",
				remote, tempPath, remoteDir, tempPath))
	} else {
		err = runner.run(fmt.Sprintf("sudo mv %s %s", tempPath, remoteDir))
	}
	return err
}

func isDir(localPath string) bool {
	fi, err := os.Stat(localPath)
	if err != nil {
		log.Fatalf("could not read file path: %v", err)
	}
	return fi.IsDir()
}

func (runner *SSHRunner) Close() {
	fmt.Printf("[DEBUG] closing connection with %s\n", runner.Host)
	runner.cancelFunc()
	err := <-runner.muxErr
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
