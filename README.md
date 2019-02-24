# ky2tf - Kubernetes YAML to Terraform HCL converter

This tool converts Kubernetes API Objects (in YAML format) into HashiCorp's Terraform configuration language.
The converted `.tf` files are suitable for use with the [Terraform Kubernetes Provider](https://www.terraform.io/docs/providers/kubernetes/index.html)
