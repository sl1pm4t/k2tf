# k2tf - Kubernetes YAML to Terraform HCL converter

This tool converts Kubernetes API Objects (in YAML format) into HashiCorp's Terraform configuration language.

The converted `.tf` files are suitable for use with the [Terraform Kubernetes Provider](https://www.terraform.io/docs/providers/kubernetes/index.html)

## Installation

## Example Usage

**Convert a single YAML file and write generated Terraform config to Stdout**

```
$ k2tf -f test-fixtures/service.yaml
```

**Convert a single YAML file and write output to file**

```
$ k2tf -f test-fixtures/service.yaml -o service.tf
```

**Convert a directory of Kubernetes YAML files**

```
$ k2tf -f test-fixtures/
```

**Read & convert Kubernetes objects directly from a cluster**

```
$ kubectl get deployments -o yaml | ./k2tf -o deployments.tf
```

## Building

> **NOTE** Requires a working Golang build environment.

This project uses Golang modules for dependency management, so it can be cloned outside of the `$GOPATH`.

**Clone the repository**

```
$ git clone https://github.com/sl1pm4t/k2tf.git
```

**Build**

```
$ cd k2tf
$ make build
```

**Run Tests**

```
$ make test
```
