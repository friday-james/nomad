package provisioning

import (
	"fmt"
	"path/filepath"
)

func deploy(target *ProvisioningTarget) error {
	var err error
	runner := target.runner
	deployment := target.Deployment

	err = runner.Open()
	if err != nil {
		return err
	}
	defer runner.Close()

	if deployment.RemoteBinaryPath == "" {
		return fmt.Errorf("remote binary path not set")
	}
	if deployment.NomadLocalBinary != "" {
		err = runner.Copy(
			deployment.NomadLocalBinary,
			deployment.RemoteBinaryPath)
		if err != nil {
			return fmt.Errorf("copying Nomad failed: %v", err)
		}
	} else if deployment.NomadSha != "" {
		s3_url := fmt.Sprintf("s3://nomad-team-test-binary/builds-oss/nomad_%s_%s.tar.gz",
			deployment.Platform, deployment.NomadSha,
		)
		remoteDir := filepath.Dir(deployment.RemoteBinaryPath)
		script := fmt.Sprintf(`aws s3 cp %s nomad.tar.gz
			sudo tar -zxvf nomad.tar.gz -C %s
			sudo chmod 0755 %s
			sudo chown root:root %s`,
			s3_url, remoteDir, deployment.RemoteBinaryPath, deployment.RemoteBinaryPath)
		err = runner.Run(script)
		if err != nil {
			return err
		}
	} else if deployment.NomadVersion != "" {
		url := fmt.Sprintf("https://releases.hashicorp.com/nomad/%s/nomad_%s_%s.zip",
			deployment.NomadVersion, deployment.NomadVersion, deployment.Platform,
		)
		remoteDir := filepath.Dir(deployment.RemoteBinaryPath)
		script := fmt.Sprintf(`curl -L --fail -o /tmp/nomad.zip %s
			sudo unzip -o /tmp/nomad.zip -d %s
			sudo chmod 0755 %s
			sudo chown root:root %s`,
			url, remoteDir, deployment.RemoteBinaryPath, deployment.RemoteBinaryPath)
		err = runner.Run(script)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no Nomad deployment specified")
	}

	for _, bundle := range deployment.Bundles {
		err = runner.Copy(
			bundle.Source, bundle.Destination)
		if err != nil {
			return fmt.Errorf("copying bundle '%s' failed: %v", bundle.Source, err)
		}
	}
	for _, step := range deployment.Steps {
		err = runner.Run(step)
		if err != nil {
			return fmt.Errorf("deployment step %q failed: %v", step, err)
		}
	}
	return nil
}
