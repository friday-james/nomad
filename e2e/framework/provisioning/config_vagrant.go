package provisioning

import "log"

// ProvisionerConfigVagrant targets a single-node Vagrant environment.
func ProvisionerConfigVagrant(config ProvisionerConfig) *ProvisioningTargets {

	if config.NomadVersion == "" && config.NomadSha == "" && config.NomadLocalBinary == "" {
		log.Fatal("cannot run vagrant provisioning without a '-nomad.*' flag set")
		return nil
	}

	return &ProvisioningTargets{
		Servers: []*ProvisioningTarget{
			&ProvisioningTarget{
				Runner: map[string]interface{}{}, // unused
				runner: &SSHRunner{
					Key:  "../.vagrant/machines/linux-ui/virtualbox/private_key",
					User: "vagrant",
					Host: "127.0.0.1",
					Port: 2222,
				},
				Deployment: Deployment{
					NomadLocalBinary: config.NomadLocalBinary,
					NomadSha:         config.NomadSha,
					NomadVersion:     config.NomadVersion,
					RemoteBinaryPath: "/opt/gopath/bin/nomad",
					Platform:         "linux_amd64",
					Bundles: []Bundle{
						Bundle{
							Source:      "../bin", // TODO: ?
							Destination: "/home/vagrant/bin",
						},
						Bundle{
							Source:      "../bin/config.hcl", // TODO: ?
							Destination: "/home/vagrant/config.hcl",
						},
					},
					Steps: []string{
						"sudo systemctl restart consul",
						"sudo systemctl restart nomad",
					},
				},
			},
		},
	}
}
