package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-providers/terraform-provider-kubernetes/kubernetes"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	configMapYAML             string
	basicDeploymentYAML       string
	deploymentYAML            string
	deployment2ContainersYAML string
	podNodeExporterYAML       string
	replicationControllerYAML string
	roleYAML                  string
	roleBindingYAML           string
	serviceYAML               string
	statefulSetYAML           string
)

func init() {
	configMapYAML = loadTestFile("configMap.yaml")
	basicDeploymentYAML = loadTestFile("basicDeployment.yaml")
	deploymentYAML = loadTestFile("deployment.yaml")
	deployment2ContainersYAML = loadTestFile("deployment2Containers.yaml")
	podNodeExporterYAML = loadTestFile("podNodeExporter.yaml")
	replicationControllerYAML = loadTestFile("replicationController.yml")
	roleYAML = loadTestFile("role.yaml")
	roleBindingYAML = loadTestFile("roleBinding.yaml")
	serviceYAML = loadTestFile("service.yaml")
	statefulSetYAML = loadTestFile("statefulSet.yaml")
}

func loadTestFile(filename string) string {
	pwd, _ := os.Getwd()
	content, err := ioutil.ReadFile(filepath.Join(pwd, "test-fixtures", filename))
	if err != nil {
		log.Fatal(err)
	}

	yaml := string(content)
	return yaml
}

func TestWriteObject(t *testing.T) {
	type args struct {
		yaml         string
		hcl          string
		resourceType string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"BasicDeployment",
			args{
				basicDeploymentYAML,
				basicDeploymentHCL,
				"kubernetes_deployment",
			},
		},
		{
			"ConfigMap",
			args{
				configMapYAML,
				configMapHCL,
				"kubernetes_config_map",
			},
		},
		{
			"Deployment",
			args{
				deploymentYAML,
				deploymentHCL,
				"kubernetes_deployment",
			},
		},
		{
			"Deployment_2Containers",
			args{
				deployment2ContainersYAML,
				deployment2ContainersHCL,
				"kubernetes_deployment",
			},
		},
		{
			"PodVolumesOnly",
			args{
				podVolumesOnlyYAML,
				podVolumesOnlyHCL,
				"kubernetes_pod",
			},
		},
		{
			"PodNodeExporter",
			args{
				podNodeExporterYAML,
				podNodeExporterHCL,
				"kubernetes_pod",
			},
		},
		{
			"role",
			args{
				roleYAML,
				roleHCL,
				"kubernetes_role",
			},
		},
		{
			"roleBinding",
			args{
				roleBindingYAML,
				roleBindingHCL,
				"kubernetes_role_binding",
			},
		},
		{
			"Service",
			args{
				serviceYAML,
				serviceHCL,
				"kubernetes_service",
			},
		},
		{
			"StatefulSet",
			args{
				statefulSetYAML,
				statefulSetHCL,
				"kubernetes_stateful_set",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := mustParseTestYAML(tt.args.yaml)
			f := hclwrite.NewEmptyFile()
			WriteObject(obj, f.Body())

			expectedConfig := parseResourceHCL(t, []byte(tt.args.hcl))
			actualConfig := parseResourceHCL(t, f.Bytes())
			assert.Equal(t, expectedConfig, actualConfig, "resource config should be equal")

			assert.True(t, validateTerraformConfig(t, tt.args.resourceType, actualConfig), "HCL should pass provider validation")
		})
	}
}

func validateTerraformConfig(t *testing.T, resourceType string, cfg *config.RawConfig) bool {
	// extract our resources rawConfig
	rsrcConfig := terraform.NewResourceConfig(cfg)

	// validate against the Kubernetes provider
	prov := kubernetes.Provider().(*schema.Provider)
	_, errs := prov.ValidateResource(resourceType, rsrcConfig)

	if len(errs) > 0 {
		// log validation errors
		for i, v := range errs {
			t.Errorf("Validation Error: %d> %v\n", i, v)
		}

		return false
	}

	return true
}

func parseResourceHCL(t *testing.T, hcl []byte) *config.RawConfig {
	// write HCL to temp location where Terraform can load it
	tmpDir, err := ioutil.TempDir("", "ky2tf")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the file
	ioutil.WriteFile(filepath.Join(tmpDir, "hcl.tf"), hcl, os.ModePerm)

	// Invoke terraform to load config
	cfg, err := config.LoadDir(tmpDir)
	if err != nil {
		t.Error(err)
	}

	// extract our resources rawConfig
	return cfg.Resources[0].RawConfig
}

func mustParseTestYAML(s string) runtime.Object {
	r := strings.NewReader(s)
	objs, err := ParseK8SYAML(r)
	if err != nil {
		panic(err)
	}
	return objs[0]
}

const configMapHCL = `resource "kubernetes_config_map" "foo_config_map" {
  metadata {
    name      = "foo-config-map"
    namespace = "bar"
    labels {
      lbl1 = "somevalue"
      lbl2 = "another"
    }
  }
  data {
    item1 = "wow"
    item2 = "wee"
  }
}
`

const basicDeploymentHCL = `resource "kubernetes_deployment" "baz_app" {
  metadata {
    name      = "baz-app"
    namespace = "bat"
    annotations {
      foo = "fam"
    }
  }
  spec {
    replicas = 2
    selector {
      match_labels {
        app = "nginx"
      }
    }
    template {
      metadata {
        labels {
          app = "nginx"
        }
        annotations {
          foo = "fam"
        }
      }
      spec {
        container {
          name  = "nginx"
          image = "nginx"
          args  = ["--debug", "--test"]
          port {
            container_port = 80
          }
          resources {
            limits {
              cpu    = "1"
              memory = "1Gi"
            }
            requests {
              cpu    = "1"
              memory = "512Mi"
            }
          }
        }
      }
    }
  }
}
`

const deploymentHCL = `resource "kubernetes_deployment" "backend_api" {
  metadata {
    name      = "backend-api"
    namespace = "default"
    labels {
      app = "backend-api"
    }
  }
  spec {
    replicas = 4
    selector {
      match_labels {
        app = "backend-api"
      }
    }
    template {
      metadata {
        labels {
          app = "backend-api"
        }
        annotations {
          "prometheus.io/port"   = "8080"
          "prometheus.io/scheme" = "http"
          "prometheus.io/scrape" = "true"
        }
      }
      spec {
        volume {
          name = "backend-api-config"
          config_map {
            name = "backend-api"
            items {
              key  = "backend-api.yml"
              path = "backend-api.yml"
            }
            default_mode = 420
          }
        }
        volume {
          name = "nginx-ssl"
          secret {
            secret_name  = "nginx-ssl"
            default_mode = 420
          }
        }
        container {
          name  = "esp"
          image = "gcr.io/endpoints-release/endpoints-runtime:1"
          args  = ["--ssl_port", "443", "--backend", "127.0.0.1:8080", "--service", "backend-api.endpoints.project.cloud.goog", "--version", "2018-11-14r0"]
          port {
            container_port = 443
            protocol       = "TCP"
          }
          volume_mount {
            name       = "nginx-ssl"
            read_only  = true
            mount_path = "/etc/nginx/ssl"
          }
          liveness_probe {
            tcp_socket {
              port = "443"
            }
            initial_delay_seconds = 5
            timeout_seconds       = 1
            period_seconds        = 10
            success_threshold     = 1
            failure_threshold     = 3
          }
          readiness_probe {
            tcp_socket {
              port = "443"
            }
            initial_delay_seconds = 5
            timeout_seconds       = 1
            period_seconds        = 10
            success_threshold     = 1
            failure_threshold     = 3
          }
          termination_message_path = "/dev/termination-log"
          image_pull_policy        = "IfNotPresent"
        }
        container {
          name    = "api"
          image   = "gcr.io/project/backend-api:0.3.15"
          command = ["/root/backend-api", "--config", "/backend-api-config/backend-api.yml", "--port", "8080", "--nats-addr=nats-streaming:4222"]
          port {
            container_port = 8080
            protocol       = "TCP"
          }
          env {
            name  = "CONF_MD5"
            value = "bedba4b80a982b3116dfd56366de3c2d"
          }
          resources {
            limits {
              cpu    = "2"
              memory = "8Gi"
            }
            requests {
              cpu    = "300m"
              memory = "2Gi"
            }
          }
          volume_mount {
            name       = "backend-api-config"
            mount_path = "/backend-api-config"
          }
          liveness_probe {
            tcp_socket {
              port = "8080"
            }
            initial_delay_seconds = 5
            timeout_seconds       = 1
            period_seconds        = 10
            success_threshold     = 1
            failure_threshold     = 3
          }
          readiness_probe {
            tcp_socket {
              port = "8080"
            }
            initial_delay_seconds = 5
            timeout_seconds       = 1
            period_seconds        = 10
            success_threshold     = 1
            failure_threshold     = 3
          }
          termination_message_path = "/dev/termination-log"
          image_pull_policy        = "Always"
        }
        restart_policy                   = "Always"
        termination_grace_period_seconds = 30
        dns_policy                       = "ClusterFirst"
      }
    }
    strategy {
      type = "RollingUpdate"
      rolling_update {
        max_unavailable = "25%"
        max_surge       = "25%"
      }
    }
    revision_history_limit    = 10
    progress_deadline_seconds = 600
  }
}
`

const deployment2ContainersHCL = `resource "kubernetes_deployment" "backend_api" {
  metadata {
    name = "backend-api"
  }
  spec {
    selector {
      match_labels {
        app = "backend-api"
      }
    }
    template {
      metadata {
        labels {
          app = "backend-api"
        }
      }
      spec {
        container {
          name  = "esp"
          image = "gcr.io/container1/image:latest"
          args  = ["--ssl_port", "443"]
          port {
            container_port = 443
            protocol       = "TCP"
          }
        }
        container {
          name    = "api"
          image   = "gcr.io/container2/image:latest"
          command = ["/root/backend-api"]
        }
      }
    }
  }
}
`

const podNodeExporterHCL = `resource "kubernetes_pod" "node_exporter_7fth_7" {
  metadata {
    name          = "node-exporter-7fth7"
    generate_name = "node-exporter-"
    namespace     = "prometheus"
    labels {
      controller-revision-hash = "2418008739"
      name                     = "node-exporter"
      pod-template-generation  = "1"
    }
    annotations {
      "prometheus.io/port"   = "9100"
      "prometheus.io/scheme" = "http"
      "prometheus.io/scrape" = "true"
    }
  }
  spec {
    volume {
      name = "default-token-rkd4g"
      secret {
        secret_name  = "default-token-rkd4g"
        default_mode = 420
      }
    }
    container {
      name  = "prom-node-exporter"
      image = "prom/node-exporter"
      port {
        name           = "metrics"
        container_port = 9100
        protocol       = "TCP"
      }
      volume_mount {
        name       = "default-token-rkd4g"
        read_only  = true
        mount_path = "/var/run/secrets/kubernetes.io/serviceaccount"
      }
      termination_message_path = "/dev/termination-log"
      image_pull_policy        = "Always"
      security_context {
        privileged = true
      }
    }
    restart_policy                   = "Always"
    termination_grace_period_seconds = 30
    dns_policy                       = "ClusterFirst"
    service_account_name             = "default"
    node_name                        = "gke-cloudlogs-dev-default-pool-4a2a9dae-9b01"
    host_pid                         = true
  }
}
`

const podVolumesOnlyHCL = `resource "kubernetes_pod" "pod_volumes_only" {
  metadata {
    name = "pod-volumes-only"
  }
  spec {
    volume {
      name = "default-token-rkd4g"
      secret {
        secret_name  = "default-token-rkd4g"
        default_mode = 420
      }
    }
    volume {
      name = "some-volume"
      config_map {
        name         = "cm1"
        default_mode = 420
      }
    }
  }
}
`

const roleHCL = `resource "kubernetes_role" "elasticsearch" {
  metadata {
    name = "elasticsearch"
  }
  rule {
    verbs      = ["get"]
    api_groups = [""]
    resources  = ["endpoints"]
  }
}
`

const roleBindingHCL = `resource "kubernetes_role_binding" "elasticsearch" {
  metadata {
    name = "elasticsearch"
  }
  subject {
    kind      = "ServiceAccount"
    name      = "elasticsearch"
    namespace = "default"
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "elasticsearch"
  }
}
`

const podVolumesOnlyYAML = `apiVersion: v1
kind: Pod
metadata:
  name: pod-volumes-only
spec:
  volumes:
  - name: default-token-rkd4g
    secret:
      defaultMode: 420
      secretName: default-token-rkd4g
  - name: some-volume
    configMap:
      defaultMode: 420
      name: cm1
`

const serviceHCL = `resource "kubernetes_service" "nginx" {
  metadata {
    name = "nginx"
    labels {
      app = "nginx"
    }
  }
  spec {
    port {
      name = "web"
      port = 80
    }
    selector {
      app = "nginx"
    }
    cluster_ip = "None"
  }
}
`

const statefulSetHCL = `resource "kubernetes_stateful_set" "web" {
  metadata {
    name = "web"
    labels {
      app = "nginx"
    }
  }
  spec {
    replicas = 14
    selector {
      match_labels {
        app = "nginx"
      }
    }
    template {
      metadata {
        labels {
          app = "nginx"
        }
      }
      spec {
        container {
          name  = "nginx"
          image = "k8s.gcr.io/nginx-slim:0.8"
          port {
            name           = "web"
            container_port = 80
          }
          volume_mount {
            name       = "www"
            mount_path = "/usr/share/nginx/html"
          }
        }
      }
    }
    volume_claim_template {
      metadata {
        name = "www"
      }
      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests {
            storage = "1Gi"
          }
        }
        storage_class_name = "thin-disk"
      }
    }
    service_name = "nginx"
  }
}
`
