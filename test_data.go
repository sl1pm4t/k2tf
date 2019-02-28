package main

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
