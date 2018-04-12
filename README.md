# Jenkins Idler [![Build Status](https://ci.centos.org/buildStatus/icon?job=devtools-fabric8-jenkins-idler-build-master)](https://ci.centos.org/job/devtools-fabric8-jenkins-idler-build-master/) [![Build Status](https://travis-ci.org/fabric8-services/fabric8-jenkins-idler.svg?branch=master)](https://travis-ci.org/fabric8-services/fabric8-jenkins-idler)


<!-- MarkdownTOC -->

- [What is it?](#what-is-it)
- [How to build](#how-to-build)
  - [Prerequisites](#prerequisites)
  - [Make usage](#make-usage)
    - [Compile the code](#compile-the-code)
    - [Build the container image](#build-the-container-image)
    - [Run the tests](#run-the-tests)
    - [Format the code](#format-the-code)
    - [Check commit message format](#check-commit-message-format)
    - [Clean up](#clean-up)
  - [Dependency management](#dependency-management)
  - [Continuous Integration](#continuous-integration)
  - [Running locally](#running-locally)
- [Misc](#misc)
- [How to contribute?](#how-to-contribute)

<!-- /MarkdownTOC -->

<a name="what-is-it"></a>
# What is it?

The Jenkins Idler is a service which idles perspectively unidles a tenant's Jenkins instance.
In order to determine whether a Jenkins instance can be idled, the Idler monitors OpenShift Build and DeploymentConfig changes.
It also keeps track of direct access to the UI as well as GitHub webhook deliveries.

![Idler Architecture](https://docs.google.com/drawings/d/e/2PACX-1vRht1rgNES66f729QUcN5oGSxtTSGVgUL_8r_c-K_Jr-iK0FWeHDak5I32l1yMiY-tN-nqQhIRYvo1G/pub?w=426&h=441)

1. Idler watches Build and DeploymentConfig changes in OpenShift
2. Idler controls the state of Jenkins DeploymentConfig in OpenShift
3. Idler is checking Jenkins Proxy for number of buffered webhook requests and last access to Jenkins UI
4. Proxy caches webhook requests while Jenkins is un-idling

Jenkins Idler is the sister project to [fabric8-jenkins-proxy](https://github.com/fabric8-services/fabric8-jenkins-proxy)(Jenkins Proxy).

<a name="how-to-build"></a>
# How to build

The following paragraphs describe how to build and work with the source.

<a name="prerequisites"></a>
## Prerequisites

The project is written in [Go](https://golang.org/), so you will need a working Go installation (Go version >= 1.8.3).

The build itself is driven by GNU [Make](https://www.gnu.org/software/make/) which also needs to be installed on your systems.

Last but not least, you need a running Docker daemon, since the final build artifact is a Docker container. Also of the unit tests make use of Docker.

<a name="make-usage"></a>
## Make usage

<a name="compile-the-code"></a>
### Compile the code

   $ make build

<a name="build-the-container-image"></a>
### Build the container image

   $ make image

<a name="run-the-tests"></a>
### Run the tests

   $ make test

<a name="format-the-code"></a>
### Format the code

   $ make fmt

<a name="check-commit-message-format"></a>
### Check commit message format

   $ make validate_commits

<a name="clean-up"></a>
### Clean up

   $ make clean

More help is provided by `make help`.

<a name="dependency-management"></a>
## Dependency management

The dependencies of the project are managed by [Dep](https://github.com/golang/dep).
To add or change the current dependencies you need to delete the Dep lock file (_Gopkg.lock_), update the dependency list (_Gopkg.toml_) and then regenerate the lock file.
The process looks like this:

    $ make clean
    $ rm Gopkg.lock
    # Update Gopkg.toml with the changes to the dependencies
    $ make build
    $ git add Gopkg.toml Gopkg.lock
    $ git commit

<a name="continuous-integration"></a>
## Continuous Integration

At the moment Travis CI and CentOS CI are configured.
Both CI systems build all merges to master as well as pull requests.

| CI System |   |
|-----------|---|
| CentOS CI | [master](https://ci.centos.org/job/devtools-fabric8-jenkins-idler-build-master/), [pr](https://ci.centos.org/job/devtools-fabric8-jenkins-idler/)|
| Travis CI | [master](https://travis-ci.org/fabric8-services/fabric8-jenkins-idler/), [pr](https://travis-ci.org/fabric8-services/fabric8-jenkins-idler/pull_requests)|

<a name="running-locally"></a>
## Running locally

The repository contains a script [`setupLocalIdler.sh`](./scripts/setupLocalIdler.sh) which can be used to run the Idler locally.

A prerequisite for this is to access the Red Hat OpenShift.io prodpreview infrastructure.

The internal documentation for how to set this up is located in this (private) [document](https://docs.google.com/document/d/1h7PIOBwtVyFl5mRuERFRL8dXBT9UMtLZdXR0Sgy-ARo/edit#heading=h.nqojkv5m23p8).

<a name="misc"></a>
# Misc

* The original [problem statement](./docs/problem-statement.md).
* [Service operations](https://docs.google.com/document/d/14rKA_Uxve5f_mFNK4vhKhXrMcquiy25AQ5tpHJQwtbc/edit#heading=h.x2mo7jq5mjcz) within OpenShift.io.

<a name="how-to-contribute"></a>
# How to contribute?

If you want to contribute, make sure to follow the [contribution guidelines](./CONTRIBUTING.md) when you open issues or submit pull requests.

<a name="apis"></a>
# APIs
Below area sample API requests

1.

    Task: Get API URL of all openshift clusters
    
    Request: http://localhost:8080/api/idler/cluster

    Response: 
    [
      {
        "APIURL": "https://api.starter-us-east-2a.openshift.com/",
        "AppDNS": "b542.starter-us-east-2a.openshiftapps.com"
      },
      {
        "APIURL": "https://api.free-stg.openshift.com/",
        "AppDNS": "1b7d.free-stg.openshiftapps.com"
      }
    ]
    
2.

    Task: Get internal state of a specified namespace

    Request: http://localhost:8080/api/idler/info/ksagathi-preview

    Response:
    {
      "Name": "ksagathi-preview",
      "ID": "7219a11c-f86a-4db1-ab3e-83216ff53009",
      "ActiveBuild": {
        "metadata": {
          "annotations": {
            
          },
          "Generation": 0
        },
        "status": {
          "phase": "New",
          "startTimestamp": "0001-01-01T00:00:00Z",
          "completionTimestamp": "0001-01-01T00:00:00Z"
        },
        "spec": {
          "replicas": 0,
          "Strategy": {
            "Type": ""
          }
        }
      },
      "DoneBuild": {
        "metadata": {
          "name": "sample1-1",
          "namespace": "ksagathi-preview",
          "annotations": {
            "openshift.io/build.number": "1",
            "openshift.io/jenkins-namespace": "ksagathi-preview-jenkins"
          },
          "Generation": 0
        },
        "status": {
          "phase": "Complete",
          "startTimestamp": "2018-04-11T08:27:15Z",
          "completionTimestamp": "2018-04-11T08:31:51Z"
        },
        "spec": {
          "replicas": 0,
          "Strategy": {
            "Type": "JenkinsPipeline"
          }
        }
      },
      "JenkinsLastUpdate": "2018-04-11T09:41:57Z"
    }


3. 

    Task: Check whether Jenkins pod is idle or not for a specific namespace
    
    Request: curl http://localhost:8080/api/idler/isidle/ksagathi-preview-jenkins?openshift_api_url=https://api.starter-us-east-2a.openshift.com/

    Response: {"is_idle":true}

4.  
  
    Task: Unidle Jenkins Pod of the a specified namespace 
    
    Request: curl http://localhost:8080/api/idler/unidle/ksagathi-preview-jenkins?openshift_api_url=https://api.starter-us-east-2a.openshift.com/

    Response: (Empty response with 200 status code)

5. 

    Task: Idle Jenkins Pod of a specified namespace

    Request: curl -i http://localhost:8080/api/idler/idle/ksagathi-preview-jenkins?openshift_api_url=https://api.starter-us-east-2a.openshift.com/

    Response: (Empty Response with 200 status code)