kontainer-engine
================

A tool like docker-machine to provision kubernetes cluster for different cloud providers.

## Building

The project includes multiple scripts in order to provide kontainer-engine binaries for multiple platforms.
The output of all scripts can be found in `./bin`.

### Linux binary

```shell
make
```

### Development native binary

```shell
./scripts/build_native_test
```

## Usage

`kontainer-engine create --driver $driverName [OPTIONS] cluster-name`

`kontainer-engine inspect cluster-name`

`kontainer-engine ls`

`kontainer-engine update [OPTIONS] cluster-name`

`kontainer-engine rm cluster-name`

To see what driver create options it has , run
`kontainer-engine create --driver $driverName --help`

To see what update options for a cluster , run
`kontainer-engine update --help cluster-ame`

A serviceAccountToken which binds to the clusterAdmin is automatically created for you, to see what it is, run
`kontainer-engine inspect clusterName`

Currently, the following drivers are supported:

 - [Google Kubernetes Engine](https://cloud.google.com/container-engine/)
 - [Amazon Elastic Container Service for Kubernetes](https://aws.amazon.com/eks/)
 - [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/)


### Google Compute Engine Usage

Driver name: gke

#### Acquiring credentials

##### Personal account

Your personal Google account can be used with kontainer-machine you just need to obtain a token:

```shell
gcloud auth login
```

##### Service account

You can download your existing Google Cloud service account file from the [Google Cloud Console](https://console.cloud.google.com/apis/credentials/serviceaccountkey), or you can create a new one from the same page.

The environment variable `GOOGLE_APPLICATION_CREDENTIALS` will be used for obtaining your credentials. It should be the absolute path to your service account file. 

If no credentials are specified, kontainer-machine will fall back to using the [Google Application Default Credentials](https://cloud.google.com/docs/authentication/production). If you are running kontainer-machine from a GCE instance, see [Creating and Enabling Service Accounts for Instances](https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances) for details.

Alternatively, you can specify credentials with `--gke-credential-path` flag.

#### Example run

```shell
kontainer-engine create --driver gke --project-id project-id --gke-credential-path /path/to/credential.json cluster-name
```

### Amazon Elastic Container Service for Kubernetes Usage

Driver name: EKS

#### Acquiring credentials

Currently the EKS driver only supports authentication via static credentials.

Static credentials can be obtained by creating a new user on your [AWS IAM console](https://console.aws.amazon.com/iam/home).

Your AWS access key and secret key can be passed as the `--client-id` and `client-secret` options respectfully.

#### Example run

```shell
kontainer-engine create --driver eks --client-id aws-access-key --client-secret aws-secret-key cluster-name
```

### Azure Kubernetes Service Usage

Driver name: AKS

#### Acquiring credentials

Authentication to AKS is done via a Service Principal. A Service Principal is an application within Azure Active Directory whose authentication tokens can be used as the client_id, client_secret and tenant_id fields needed by kontainer-machine.

The easiest way to create a new Service Principal is using the Azure CLI.

First off, login to your account:

```shell
az login
```

Get your current subscription id:

```shell
export SUBSCRIPTION_ID="$(az account list | jq -r ".[] | .id" | head -n 1)"
```

Configure the subscription id for Azure CLI:

```shell
az account set --subscription="$SUBSCRIPTION_ID"
```

Finally, the Service Principal can be created:

```shell
az ad sp create-for-rbac --role="Contributor" --scopes="/subscriptions/$SUBSCRIPTION_ID"
```

This command outputs something similar to the following:

```json
{
  "appId": "00000000-0000-0000-0000-000000000000",
  "displayName": "azure-cli-2017-06-05-10-41-15",
  "name": "http://azure-cli-2017-06-05-10-41-15",
  "password": "0000-0000-0000-0000-000000000000",
  "tenant": "00000000-0000-0000-0000-000000000000"
}
```

These values make to the kontainer-machine options like so:

 - client-id is the appId defined above.
 - client-secret is the password defined above.
 - tenant-id is the tenant defined above

#### Example run

```shell
kontainer-engine create --driver aks --client-id appId --client-secret password --tenant-id tenant --subscription-id subscription-id --resource-group resource-group --public-key ~/.ssh/id_rsa.pub cluster-name
```

## Running

```shell
./bin/kontainer-engine
```

## Tests

Run tests with:

```shell
./run_integration_tests
```

You must have Go and [Bats](https://github.com/sstephenson/bats) installed for the tests to run.

If you are adding new tests, note that they must have a `.bats` extension to be recognized by the runner.

## License
Copyright (c) 2014-2018 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
