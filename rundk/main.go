package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"knative.dev/test-infra/pkg/helpers"
	"knative.dev/test-infra/pkg/interactive"
)

func main() {
	image := flag.String("test-image", "gcr.io/knative-tests/test-infra/prow-tests:stable", "The image we use to run the test flow.")
	mounts := flag.String("mounts", "", "A list of extra folders or files separated by comma that need to be mounted to run the test flow.")
	mandatoryEnvVars := flag.String("mandatory-env-vars", "GOOGLE_APPLICATION_CREDENTIALS", "A list of env vars separated by comma that must be set on local.")
	optionalEnvVars := flag.String("optional-env-vars", "", "A list of env vars separated by comma that optionally need to be set on local.")
	flag.Parse()

	cmd, cancel := setup(*image, strings.Split(*mounts, ","),
		strings.Split(*mandatoryEnvVars, ","), strings.Split(*optionalEnvVars, ","))
	defer cancel()

	run(cmd, *image, flag.Args()...)
}

func setup(image string, mounts, mandatoryEnvVars, optionalEnvVars []string) (interactive.Docker, func()) {
	var err error

	builtUpDefers := make([]func(), 0)

	envs := interactive.Env{}
	if err = envs.PromoteFromEnv(mandatoryEnvVars...); err != nil {
		log.Fatalf("Missing mandatory argument: %v", err)
	}
	envs.PromoteFromEnv(optionalEnvVars...) // Optional, so don't check error

	// Setup temporary directory
	tmpDir, err := ioutil.TempDir("", "prow-docker.")
	if err != nil {
		log.Fatalf("Error setting up the temporary directory: %v", err)
	}
	fmt.Printf("Logging to %s\n", tmpDir)

	// Setup command
	cmd := interactive.NewDocker()
	cmd.LogFile = path.Join(tmpDir, "build-log.txt")
	// Mount the required files and directories on the host machine to the container
	for _, m := range mounts {
		cmd.AddMount("bind", m, m)
	}

	repoRoot, err := helpers.GetRootDir()
	if err != nil {
		log.Fatalf("Error getting the repo's root directory: %v", err)
	}
	// Mount source code dir
	// Add overlay mount over the user's git repo, so the flow doesn't mess it up
	cancel := cmd.AddRWOverlay(repoRoot, repoRoot)
	builtUpDefers = append(builtUpDefers, cancel)

	// Add overlay mount for kube context to be available (if reusing an existing cluster)
	// If future use needs other directories, mounting the whole home directory could be a pain
	//  because our upstream prow-tests will be install Go in /root/.gvm
	cancel = cmd.AddRWOverlay(path.Join(os.Getenv("HOME"), ".kube"), "/root/.kube")
	builtUpDefers = append(builtUpDefers, cancel)

	// Starting directory
	cmd.AddArgs(fmt.Sprintf("-w=%s", repoRoot))

	extArtifacts := os.Getenv("ARTIFACTS")
	// Artifacts directory
	if len(extArtifacts) == 0 {
		log.Printf("Setting local ARTIFACTS directory to %s", tmpDir)
		extArtifacts = tmpDir
	}
	cmd.AddMount("bind", extArtifacts, extArtifacts)
	envs["ARTIFACTS"] = extArtifacts
	builtUpDefers = append(builtUpDefers, func() {
		log.Printf("Artifacts found at %s", extArtifacts)
	})
	cmd.AddEnv(envs)

	// Until everyone is using 20.xx version of docker (see https://github.com/docker/cli/pull/1498) adding the --pull flag,
	//  need to separately pull the image first to be sure we have the latest
	pull := interactive.NewCommand("docker", "pull", image)
	pull.Run()

	return cmd, func() {
		for _, ff := range builtUpDefers {
			ff()
		}
	}
}

func run(cmd interactive.Docker, image string, commandAndArgsOpt ...string) error {
	// Finally add the image then command to run (if any)
	cmd.AddArgs(image)
	cmd.AddArgs("runner.sh")
	cmd.AddArgs(commandAndArgsOpt...)
	fmt.Println(cmd)
	fmt.Println("Starting in 3 seconds, ^C to abort!")
	time.Sleep(time.Second * 3)
	return cmd.Run()
}
