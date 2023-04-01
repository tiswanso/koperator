# koperator E2E Testing

## Scenarios

The e2e test scenarios are intended for testing within any Kubernetes cluster, unless otherwise indicated.

1. [Basic-sanity](#Basic-Sanity)
   - (optional) install of documented dependencies
   - install of koperator
   - creation of simple Kakfa cluster (3 brokers)
   - basic validations of KafkaCluster status and pods statuses
   - delete KafkaCluster
   - delete koperator
   - basic validation of resource cleanup
   - (optional) dependency uninstall

## Test Implementation Pattern

### pkg/kube

Client access objects and methods to interact with k8s api.  It's a package to implement wrappers for
some common actions.

### Install Profiles

Intended to cover install actions and settings that are typically linked for specific scenarios.

`test/pkg/install` defines an interface for install profiles and the package has 2 profile implementations:
1. `BasicDependenciesInstaller` -- installs the basic dependencies documented in the koperator docs
2. `BasicKafkaInstaller` -- installs the basic koperator and creates a simple KafkaCluster

## Test Implementations

### Basic-Sanity

`base_install_test.go :: TestInstall()`

Covers the steps documented in the kafka-operator basic install doc section:
https://banzaicloud.com/docs/supertubes/kafka-operator/install-kafka-operator/

#### Args:
- (required) `--kubeconfig <kconf file>` or env var setup
- (optional) `--noprereqs` don't install prereqs
- (optional) `--cleanup` test cleanup after install test

#### Tests

The implementation is built with the test/pkg/kube and test/pkg/install to install/uninstall
koperator dependencies and koperator using the pkg/install profiles.  The `BasicKafkaInstaller` profile
creates a KafkaCluster using the config samples simple KafkaCluster example.

**NOTE** The koperator dependencies are installed/uninstalled by default and can be disabled if the
`--noprereqs` is passed via the command line args.

The test checks the KafkaCluster status and broker pods statuses, waiting for them to become "active".

If the `--cleanup` flag is enabled, the uninstall tests are run.

Individual test run examples:

Run the "Check KafkaCluster" test
```
go test -v -run=TestInstall/"Check KafkaCluster" --kubeconfig /Users/tiswanso/.kube/koper.kconf
```

Run the "Uninstall Kafka" test
```
go test -v -run=TestInstall/"Uninstall Kafka" --kubeconfig /Users/tiswanso/.kube/koper.kconf --cleanup --noprereqs
```

