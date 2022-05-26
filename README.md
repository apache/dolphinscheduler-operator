# dolphinscheduler-operator


## Project Status

**Project status:** *alpha1*

**Current API version:** *`v1alpha1`*

## Prerequisites

**go version :** *go1.17.6*

**minikube version:** *v1.25.1*

**kubebuilder version:** *3.3.0*

## Get Started
1. create  namespace ds

2. install  postgres (not required)

    if had no postgressql ,you can turn into config/configmap and run *kubectl apply -f postgreSQL/* 

    connect to postgressql and run the sql script in  dolphinscheduler/dolphinscheduler-dao/resources/sql

    record the deployment ip   172.17.0.3

![image](https://user-images.githubusercontent.com/7134124/170439546-87cce0df-6cb4-4ab1-bb01-9200309efe45.png)

3. install  zookeeper(not required) 
    if had no zookeeper ,the doployment file is in config/configmap/zookeeper ,run "kubectl apply -f zookeeper/" and record the ip ,eg :172.17.0.4
