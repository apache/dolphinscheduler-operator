# dolphinscheduler-operator

## feature

1. deployment the master ,worker moudle
2. scale the pods numbers with one commond
3. update the master,worker version  quickly (not include the sql)

## Project Status

**Project status:** *'alpha1'*

**Current API version:** *`v1alpha1`*

## Prerequisites

**go version :** *go1.17.6*

**minikube version:** *v1.25.1*

**kubebuilder version:** *3.3.0*

**kubectl version:** *1.23.1*

## Get Started

1. **create  namespace ds**

    kubectl create namespace ds

2. **install  postgres (not required)**

    if had no postgressql ,you can turn into config/ds/ and run *"kubectl apply -f postgreSQL/"* ,but you need to replace your local document to hostPath.path in postgres-pv.yaml first

    connect to postgressql and run the sql script in  dolphinscheduler/dolphinscheduler-dao/resources/sql

    record the deployment ip  eg: 172.17.0.3

![image](https://user-images.githubusercontent.com/7134124/170439546-87cce0df-6cb4-4ab1-bb01-9200309efe45.png)


3. **install  zookeeper(not required)**

    if had no zookeeper ,the doployment file is in config/ds/zookeeper ,run *"kubectl apply -f zookeeper/"* and record the ip ,eg :172.17.0.4


4. **create pv and pvc (not required)**

    if you had pv and pvc ,you can config it in config/sameples

    or you can create it with config/ds/ds-pv.yaml and config/configmap/ds-pvc.yaml .notice to replace your local document address in hostPath.path in ds-pv.yaml

    and you can mount the lib in dolphinscheduler /opt/soft  in config/samples/ds_v1alpha1_dsworker.yaml with paramter named lib_pvc_name

    mount the logs in /opt/dolphinscheduler/logs with the paramters named log_pvc_name with pvcname

 ## how to test

 * replace the database config and zookeeper config paramters in config/samples/*.yaml

 * replace the nodeport in *config/samples/ds_v1alpha1_api.yaml*

 * in current project  run *"make build && make manifests && make install && make run"*

 * cd to config/samples

 * first run *"kubectl apply -f ds_v1alpha1_dsalert.yaml "*

 * then run  *"kubectl apply -f ds_v1alpha1_api.yaml -f ds_v1alpha1_dsmaster.yaml -f ds_v1alpha1_dsworker.yaml "*

 ## the result

 ![image](https://user-images.githubusercontent.com/7134124/171322789-86adfaac-57ad-4e8e-b092-8704b84d20c3.png)
