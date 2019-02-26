# k2tf - Kubernetes YAML to Terraform HCL converter

This tool converts Kubernetes API Objects (in YAML format) into HashiCorp's Terraform configuration language.

The converted `.tf` files are suitable for use with the [Terraform Kubernetes Provider](https://www.terraform.io/docs/providers/kubernetes/index.html)

## Example Usage

**Convert a single YAML file and write generated Terraform config to Stdout**

`k2tf -f test-fixtures/service.yaml`

**Convert a single YAML file and write to `.tf` file**

`k2tf -f test-fixtures/service.yaml -o service.tf`

**Convert a directory of Kubernetes YAML files**

`k2tf -f test-fixtures/`

**Read & Convert Kubernetes objects directly from a cluster**

`kubectl get deployments -o yaml | ./k2tf -o deployments.tf`
