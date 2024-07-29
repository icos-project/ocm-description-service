# OCM Description Service

## Overview

The `ocm-description-service` is a service designed to offload workloads on various orchestrators such as OCM, Nuvla, and others. This service provides a set of RESTful endpoints to execute jobs pulled from Job Manager, perform health checks, and synchronize resources efficiently.

## Table of Contents

- [1. Job Types and Subtypes](#1-job-types-and-subtypes)
- [2. Locking and Ownership Mechanism](#2-locking-and-ownership-mechanism)
- [3. Remediation Actions](#3-remediation-actions)
- [4. Deployment Management](#4-deployment-management)
- [5. Resource Status Tracking](#5-resource-status-tracking)
- [6. Docker Installation](#6-docker-installation)
- [7. Kind Installation](#7-kind-installation)
- [8. Usage](#8-usage)
- [8. Contributing](#9-contributing)
- [10. Legal](#10-legal)


## 1. Job Types and Subtypes

In the ICOS continuum, various job types and subtypes are used to manage deployment and remediation tasks effectively. Below is a brief explanation of each job type and its associated subtypes:

- `CreateDeployment`: initiates the deployment of an application.
- `DeleteDeployment`: removes an application.
- `UpdateDeployment`: updates a certain deployment.
    - **Subtypes:**
        - `ScaleIn`
        - `ScaleOut`
        - `ScaleUp`
        - `ScaleDown`
        - `SecurityRemediation`

## 2. Locking and Ownership Mechanism

Whenever an OCM Descriptor Service instance picks up a job, it sends a request to the Job Manager to "Lock" the job from being executed by other Descriptor Service instances. Additionally, "Ownership" is set to ensure that, in the context of a multicluster deployment with multiple hubs, a certain job is locked and owned by an OCM Hub. This mechanism ensures job consistency and prevents race conditions across the deployment clusters.

## 3. Remediation Actions

When the Policy Manager detects an incompliance, it sends a request to the Job Manager to create an `UpdateDeployment` job. This job is then processed by the Description Service. Currently, we support five different job subtypes to handle remediation actions:

- `ScaleIn`: Adds a replica to a deployment to handle increased load or improve redundancy.
- `ScaleOut`: Removes a replica from a deployment to reduce resource usage when demand decreases.
- `ScaleUp`: Increases a deployment's resources by adding 100 MB of memory and 100 CPU units.
- `ScaleDown`: Decreases a deployment's resources by removing 100 MB of memory and 100 CPU units.
- `SecurityRemediation`: Applies a security update to ensure the deployment adheres to the latest security standards. (TODO: Detailed implementation pending)


## 4. Deployment Management 

In the ICOS continuum, the deployment and undeployment of applications are managed through two primary job types: `CreateDeployment` and `DeleteDeployment`:

- `CreateDeployment`: this job type will create a ManifestWork in a target ManagedCluster that is specified in the specs of the job. 
- `DeleteDeployment`: this job type will remove a ManifestWork in a target ManagedCluster that is specified in the specs of the job. 


## 5. Resource Status Tracking

The OCM Descriptor Service relies on a [sidecar container](https://production.eng.it/gitlab/icos/meta-kernel/ocm-descriptor-sidecar/) responsible for scheduling. The sidecar container ensures that resource statuses are regularly updated and synchronized. 

## 6. Docker Installation

To install and run the `ocm-description-service`, follow these steps:

1. Clone the repository:
    ```sh
    git clone https://production.eng.it/gitlab/icos/meta-kernel/ocm-description-service.git
    ```

2. Navigate to the project directory:
    ```sh
    cd ocm-description-service
    ```

3. Build the Docker image:
    ```sh
    docker build -t ocm-description-service .
    ```

4. Set up the environment variable for the job manager URL.
    ```sh
    export JOBMANAGER_URL=http://localhost:8082
    ```

5. Run the Docker container:
    ```sh
    docker run -e JOBMANAGER_URL=$JOBMANAGER_URL -p 8083:8083 ocm-description-service
    ```

## 7. Kind Installation

Please, refer to the helm suite in [ICOS Agent Repository](https://production.eng.it/gitlab/icos/suites/icos-agent)

## 8. Usage

After running the service, you can use tools like `curl`, `Postman`, `Swagger` or any other API client to interact with the endpoints. Beware that you will need a Keycloak Token to perform requests to this service.

### Example Request
```sh
curl --location 'http://localhost:8083/deploy-manager/execute' \
--header 'Authorization: Bearer (Token)'
```

## 9. Contributing

In order to contribute to this repository, feel free to open a pull request and assign `@x_alvolkov`or `x_magallar` as a reviewer.

## 10. Legal

The OCM Deployment Manager is released under the Apache 2.0 license.
Copyright © 2022-2024 Eviden. All rights reserved.

🇪🇺 This work has received funding from the European Union's HORIZON research and innovation programme under grant agreement No. 101070177.
