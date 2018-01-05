#!/usr/bin/env bats

load ../assertions

setup() {
    AZURE_SUBSCRIPTION_ID=my-subscription-id

    # load mock azure service definition
    hoverctl import ./integration-tests/aks/azure-api.json
}

#########################
# TEST VALIDATIONS WORK #
#########################
@test "create should require a name" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks

  output_contains "Cluster name is required"
  [ "$contains" = true ]
}

@test "create should require a resource group" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks my-super-cluster-name

  output_contains "resource group is required"
  [ "$contains" = true ]
}

@test "create should require a path to a public key" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube my-super-cluster-name

  output_contains "path to ssh public key is required"
  [ "$contains" = true ]
}

@test "create should require a admin username" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub my-super-cluster-name

  output_contains "admin username is required"
  [ "$contains" = true ]
}

@test "create should require an agent dns prefix" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin my-super-cluster-name

  output_contains "agent dns prefix is required"
  [ "$contains" = true ]
}

@test "create should require an agent pool name" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix my-super-cluster-name

  output_contains "agent pool name is required"
  [ "$contains" = true ]
}

@test "create should require a client id" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname my-super-cluster-name

  output_contains "client id is required"
  [ "$contains" = true ]
}

@test "create should require a client secret" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname --client-id 12345 my-super-cluster-name

  output_contains "client secret is required"
  [ "$contains" = true ]
}

@test "create should require a subscription id" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname --client-id 12345 --client-secret 67890 my-super-cluster-name

  output_contains "subscription id is required"
  [ "$contains" = true ]
}

######################
# TEST START CLUSTER #
######################
@test "set up cluster" {
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname --client-id 12345 --client-secret 67890 --subscription-id 1029384857 my-super-cluster-name

  output_contains "Cluster provisioned successfully"
  [ "$contains" = true ]
}

@test "it prevents duplicate cluster names" {
  ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname --client-id 12345 --client-secret 67890 --subscription-id 1029384857 my-super-cluster-name
  run ./kontainer-engine create --base-url http://localhost:8500 --driver aks --resource-group kube --public-key ./integration-tests/test-key.pub --admin-username ohadminmyadmin --node-dns-prefix mydnsprefix --node-pool-name mypoolname --client-id 12345 --client-secret 67890 --subscription-id 1029384857 my-super-cluster-name

  output_contains "Cluster my-super-cluster-name already exists"
  [ "$contains" = true ]
}
