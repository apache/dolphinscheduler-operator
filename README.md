<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one
  ~ or more contributor license agreements.  See the NOTICE file
  ~ distributed with this work for additional information
  ~ regarding copyright ownership.  The ASF licenses this file
  ~ to you under the Apache License, Version 2.0 (the
  ~ "License"); you may not use this file except in compliance
  ~ with the License.  You may obtain a copy of the License at
  ~
  ~   http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing,
  ~ software distributed under the License is distributed on an
  ~ "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
  ~ KIND, either express or implied.  See the License for the
  ~ specific language governing permissions and limitations
  ~ under the License.
-->

# dolphinscheduler-operator

## Features

- Deploy and manage the master, worker, alert, api components.
- Scale the Pod numbers with one commond.
- Update the component's version (not include the database schema).

## Project Status

Project status: `alpha1`

Current API version: `v1alpha1`

## Get Started

- Create a namespace `ds`

```shell
kubectl create namespace ds
```

- Install PostgreSQL database (Optional)

If you don't have a running database, you can run `kubectl apply -f config/ds/postgreSQL`
to create a demo database, note that this is only for demonstration, DO NOT use it in production environment.
You need to replace the `hostPath.path` in `postgres-pv.yaml` if you don't have a directory `/var/lib/data`.

Connect to PostgreSQL and initialize the database schema by executing
[`dolphinscheduler/dolphinscheduler-dao/src/main/resources/sql/dolphinscheduler_postgresql.sql`](https://github.com/apache/dolphinscheduler/blob/dev/dolphinscheduler-dao/src/main/resources/sql/dolphinscheduler_postgresql.sql).

- Install zookeeper (Optional)

If you don't have a running zookeeper, the demo doployment file is in `config/ds/zookeeper`,
run `kubectl apply -f config/ds/zookeeper`.

- Create pv and pvc (Optional)

If you have pv and pvc, you can config it in `config/sameples`.

Or you can create it with `config/ds/ds-pv.yaml` and `config/configmap/ds-pvc.yaml`.
Notice to replace the `hostPath.path` in `ds-pv.yaml`.

And you can mount the lib in dolphinscheduler `/opt/soft`  in config/samples/ds_v1alpha1_dsworker.yaml with paramter named lib_pvc_name

Mount the logs in `/opt/dolphinscheduler/logs` with the pvcname named `log_pvc_name`.

## how to test

* Replace the database config and zookeeper config paramters in [`config/samples/`](./config/samples/).

* Replace the nodeport in [`config/samples/ds_v1alpha1_api.yaml`](./config/samples/ds_v1alpha1_dsapi.yaml)

* Install CRDs and controller

```shell
export IMG=ghcr.io/apache/dolphinscheduler-operator:latest
make build && make manifests && make install && make deploy
```

* Deploy the sample

```shell
cd config/samples
kubectl apply -f ds_v1alpha1_dsalert.yaml
kubectl apply -f ds_v1alpha1_api.yaml -f ds_v1alpha1_dsmaster.yaml -f ds_v1alpha1_dsworker.yaml
```
