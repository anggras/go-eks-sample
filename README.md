# Example Go EKS Setup

This repository provides a sample Go application deployment using EKS.

## Table of Contents:
- [Configuring AWS Credentials for `eksctl`](#configuring-aws-credentials-for-eksctl)
- [Setting Up EKS Cluster](#setting-up-eks-cluster)
- [Deploying App to the Cluster](#deploying-app-to-the-cluster)
- [Create IAM OIDC Identity Provider for the Cluster](#create-iam-oidc-identity-provider-for-the-cluster)
- Others
  - [Scaling](#scaling)
  - [Specifying Resource (CPU & Memory)](#specifying-resource-cpu--memory)
  - [Log Collection and Monitoring](#log-collection-and-monitoring)
  - [Setting Up Alarm](#setting-up-alarm)
  - [Sharing Application Load Balancer (ALB)](#sharing-application-load-balancer-alb)

## Tools
Useful tools to manage EKS:
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)
- [eksctl](https://eksctl.io) 
- [AWS Command Line Interface (CLI)](https://aws.amazon.com/cli/)

## Configuring AWS Credentials for `eksctl`
The usual setup for AWS CLI also works for `eksctl`, utilising the `~/.aws/credentials` file or [environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html) as explained in https://eksctl.io/installation/#prerequisite. 

You can also use the `--profile` flag to specify the profile for `eksctl`.

## Setting Up EKS Cluster
You can setup a new EKS cluster with either of the following methods:
- using `eksctl`
- using AWS Management Console
- using AWS CLI

For this example, we will focus more on using `eksctl` to setup the cluster, using the following command:

```sh
eksctl create cluster --name <cluster_name>
```

This command will create a `<cluster_name>` EKS cluster with default settings, such as:
- 2 x m5.large worker nodes 
- using the official AWS EKS AMI
- us-west-2 region (or the profile's default region)
- a dedicated VPC

For more examples on creating EKS cluster with various configuration: https://eksctl.io/getting-started/ or https://eksctl.io/usage/creating-and-managing-clusters/.

## Create IAM OIDC identity provider for the cluster
```sh
eksctl utils associate-iam-oidc-provider --cluster <cluster_name> --approve
```
More details: https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html

## Installing the AWS Load Balancer Controller add-on
The complete guide is available at https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html

### Create the IAM Policy

The [iam_policy.json](iam_policy.json) is included in this project.

```sh
aws iam create-policy \
    --policy-name AWSLoadBalancerControllerIAMPolicy \
    --policy-document file://iam_policy.json
```
This only needs to be created once if never been done before

### Create the IAM Role for the service account of the cluster
```sh
eksctl create iamserviceaccount \
  --cluster=<cluster_name> \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --role-name AmazonEKSLoadBalancerControllerRole \
  --attach-policy-arn=arn:aws:iam::<aws_account_id>:policy/AWSLoadBalancerControllerIAMPolicy \
  --approve
```

### Install the AWS Load Balancer Controller using Helm
#### Add the `eks-charts` repository
```sh
helm repo add eks https://aws.github.io/eks-charts
```
#### Update local repo
```sh
helm repo update eks
```
#### Install the AWS Load Balancer Controller
```sh
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=<cluster_name> \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller 
```
#### Verify that the controller is installed
```sh
kubectl get deployment -n kube-system aws-load-balancer-controller

```
## Deploying App to the Cluster

### Creating the Elastic Container Registry (ECR) Repository

```sh
aws ecr create-repository --repository-name <app_repository_name>
```

or,

Create the repository using the AWS Management Console: https://docs.aws.amazon.com/AmazonECR/latest/userguide/repository-create.html

### Perparing the Container Image
After creating the ECR repository, you should be able to view the push commands from the AWS Management Console or by following this guide: https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html 

#### Authenticating with the Registry
First, you'll need to authenticate the Docker client with the registry

```sh
aws ecr get-login-password --region <region> | docker login --username AWS --password-stdin <aws_account_id>.dkr.ecr.<region>.amazonaws.com
```
Do not forget to replace `<aws_account_id>` and `<region>` with the actual values.

#### Building the Container Image
```sh
docker build -t <app_repository_name> .
```
Note: you may need to use `buildx build --platform=linux/amd64` to make sure it is build for the correct platform

#### Tagging the Container Image
```sh
docker tag <app_repository_name>:latest <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<app_repository_name>:latest
```
#### Pushing the Container Image to the Repository
```sh
docker push <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<app_repository_name>:latest
```

### Creating the Namespace
```sh
kubectl create namespace <namespace_name>
```
Namespace used in this example: `simple-app`

### Applying the Manifests
```sh
kubectl apply -f manifests_app/deployments.yml -f manifests_app/service_nlb.yml
```

### Getting the Load Balancer's Information 
This will get the load balancer's external IP / DNS name
```sh
kubectl get svc -n simple-app
```

## Others
### Scaling
In Kubernetes, there are 2 main mechanisms available to scale capacity automatically to maintain steady and predictable performance.

#### Scaling Compute Resources
Some possible options for scaling compute resource are:
- [Cluster Autoscaler (CA)](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/README.md), or
- [Karpenter](https://karpenter.sh/)

Tldr; the underying technology for CA (in AWS) is based on Auto Scaling Group, while Karpenter is able to launch the right-sized compute resources depending on theworkload requirements.

#### Scaling Workloads
To scale the EKS workloads, these are some of the options:
- [Horizontal Pod Autoscaler (HPA)](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/), where the number of replicas can be adjusted based on average CPU utilisation, average memory utilisation or any other custom metric.
- [Cluster Proportional Autoscaler (CPA)](https://github.com/kubernetes-sigs/cluster-proportional-autoscaler), where the replicas are scaled based on the number of nodes in a cluster. Example application: CoreDNS and other services that needs to scale according to the number of nodes in the cluster.

### Specifying Resource (CPU & Memory)
To specify the resource for your nodes, you can specify the instance type and sizes by specifying the node groups: https://eksctl.io/usage/managing-nodegroups/.

For containers and pods, you can also specify the [memory](https://kubernetes.io/docs/tasks/configure-pod-container/assign-memory-resource/) and [CPU](https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/).

### Log Collection and Monitoring
One of the easiest method to do log collection and monitoring is to enable Container Insights: https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/deploy-container-insights-EKS.html

### Setting Up Alarm
To be notified based on [certain pattern](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntaxForMetricFilters.html) in a log file, you can create a CloudWatch Metric Filter: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/MonitoringPolicyExamples.html

And then, based on that Metric Filter, you can [create a CloudWatch Alarm](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Create_alarm_log_group_metric_filter.html) which in turn can be used to trigger an SNS topic (e.g. to send message through Slack)

### Sharing Application Load Balancer (ALB)
To share an ALB across multiple namespaces, [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) can be used.

To apply ALB ingress for this example:
```sh
kubectl apply -f manifests_app/ingress.yml
```

#### Deploy a Second Application (**app2**)

Create new ECR Repository for **app2**
```sh
aws ecr create-repository --repository-name <app2_repository_name>
```
Build **app2**
```sh
docker build -f Dockerfile-app2 -t <app2_repository_name> .
```
Tag **app2** container image
```sh
docker tag <app2_repository_name>:latest <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<app2_repository_name>:latest
```
Push **app2** container image to repository
```sh
docker push <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<app2_repository_name>:latest
```
Create namespace for **app2**
```sh
kubectl create namespace <app2_namespace_name>
```
Namespace used in this example: `simple-app2`

Apply **app2** manifests
```sh
kubectl apply -f manifests_app2
```
To get the ALB DNS name, run
```sh
kubectl get ingress -n <namespace_name>
```

You should be able to access the **app** service at
```
<ALB_DNS_name>/app1
```
and **app2** service at
```
<ALB_DNS_name>/app2
```

## Workshop
AWS has a good practical workshop on EKS that you can do at your own pace, available at https://www.eksworkshop.com.