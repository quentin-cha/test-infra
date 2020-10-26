# rundk

`rundk` is a tool to run a test command from the test image, by using it
developers can reproduce the test flow as run in the CI environment.

## Installation

`rundk` can be installed and upgraded by running:

```shell
go get knative.dev/test-infra/rundk
```

## Usage

```shell
Usage of rundk:
  -test-image string
        The image we use to run the test flow. (default "gcr.io/knative-tests/test-infra/prow-tests:stable")
  -mounts string
        A list of extra folders or files separated by comma that need to be mounted to run the test flow.
  -mandatory-env-vars string
        A list of env vars separated by comma that must be set on local. (default "GOOGLE_APPLICATION_CREDENTIALS")
  -optional-env-vars string
        A list of env vars separated by comma that optionally need to be set on local.
```

### Example

Run E2E tests for a Knative repository:

```shell
rundk --mounts=/temp/gcloud-secret-key.json ./test/e2e-tests.sh
```

> Note: the `rundk` command must be run under the root or sub directory of the
> local Knative repository.
