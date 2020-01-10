package provisioning

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

// ProvisionerConfigTerraform targets a Terraform cluster by reading the config
// from a stdin, as output by 'terraform output provisioning'. Malformed inputs
// will log.Fatal so that we halt the test run.
func ProvisionerConfigTerraform(config ProvisionerConfig) *ProvisioningTargets {
	file, err := ioutil.ReadFile(config.TerraformConfig)
	if err != nil {
		log.Fatalf("could not read -provision.terraform file: %v", err)
	}

	targets := &ProvisioningTargets{}
	err = json.Unmarshal([]byte(file), &targets)
	if err != nil {
		fmt.Printf("decoding error: %v\n", err)
	}

	for _, server := range targets.Servers {
		canonicalize(server, config)
	}
	for _, client := range targets.Clients {
		canonicalize(client, config)
	}
	return targets
}

func canonicalize(target *ProvisioningTarget, config ProvisionerConfig) {

	// allow the '-nomad.*' command line flags to override
	// the values we get from 'terraform output provisioning'
	if config.NomadVersion != "" {
		target.Deployment.NomadVersion = config.NomadVersion
	}
	if config.NomadSha != "" {
		target.Deployment.NomadSha = config.NomadSha
	}
	if config.NomadLocalBinary != "" {
		target.Deployment.NomadLocalBinary = config.NomadLocalBinary
	}

	if target.Deployment.RemoteBinaryPath == "" {
		log.Fatal("bad runner config for 'remote_binary_path': missing value")
	}
	key, ok := target.Runner["key"].(string)
	if !ok {
		log.Fatalf("bad runner config for 'key': %v", target.Runner)
	}
	user, ok := target.Runner["user"].(string)
	if !ok {
		log.Fatalf("bad runner config for 'user': %v", target.Runner)
	}
	hostname, ok := target.Runner["host"].(string)
	if !ok {
		log.Fatalf("bad runner config for 'host': %v", target.Runner)
	}
	port, ok := target.Runner["port"].(float64)
	if !ok {
		log.Fatalf("bad runner config for 'port': %v", target.Runner)
	}

	target.runner = &SSHRunner{
		Key:  key,
		User: user,
		Host: hostname,
		Port: int(port),
	}
}
