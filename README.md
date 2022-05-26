# dolphinscheduler-operator

## feature 

1. deployment the master ,worker moudle 
2. scale the pods numbers with one commond 
3. update the master,worker version  quickly (not include the sql)

## Project Status

**Project status:** *alpha1*

**Current API version:** *`v1alpha1`*

## Prerequisites

**go version :** *go1.17.6*

**minikube version:** *v1.25.1*

**kubebuilder version:** *3.3.0*

**kubectl version:** *1.23.1*

## Get Started
1. **create  namespace ds**

2. **install  postgres (not required)**

    if had no postgressql ,you can turn into config/configmap and run *kubectl apply -f postgreSQL/* 

    connect to postgressql and run the sql script in  dolphinscheduler/dolphinscheduler-dao/resources/sql

    record the deployment ip  eg: 172.17.0.3

![image](https://user-images.githubusercontent.com/7134124/170439546-87cce0df-6cb4-4ab1-bb01-9200309efe45.png)

3. **install  zookeeper(not required) **

    if had no zookeeper ,the doployment file is in config/configmap/zookeeper ,run *"kubectl apply -f zookeeper/"* and record the ip ,eg :172.17.0.4

4.  **create  and merge the configmaps**

    replace the postgressql ip in application.yaml
    
    there  are four application.yaml that needed  to merge in the following locations:
    
    config/configmap/alert
    config/configmap/api
    config/configmap/master
    config/configmap/worker
    
    run  *"kubectl create cm ds-${name}-config --from-file=application.yaml -n  ds"* in these document
    
    the result is 
    
    ![image](https://user-images.githubusercontent.com/7134124/170443875-217e12e6-d50d-4ef2-b3ac-b81f4d5a7666.png)

5. **create alert moudle**

    run *"kubectl apply -f alert/"* in config/configmap/
    
6.  **create the api moudle**

    replace the zookooper ip in config/configmap/api/ds-api-deployment.yaml
    
    run *"kubectl apply -f api/"* in config/configmap/
    
7. **create pv and pvc (not required)**    
   
    if you had pv and pvc ,you can config it in config/sameples 
    
    or you can create it with config/configmap/ds-pv.yaml and config/configmap/ds-pvc.yaml
    
    and you can mount the lib in dolphinscheduler /opt/soft  in config/samples/ds_v1alpha1_dsworker.yaml with paramter named lib_pvc_name  
    
    mount the logs in /opt/dolphinscheduler/logs with the paramters named log_pvc_name with pvcname
    
 ## how to test
 
    in current project  run *"make manifests && make install && make run"* 

    in config/confgimap ,merge the *"zookeeper_connect"* with the zookeeper ip in two files .or other paramters ,all the paramters you can find  in      api/v1alpha1/ds${crdname}_types.go
    
    run *"kubectl apply -f samples"* 
     
    
    
