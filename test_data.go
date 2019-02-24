package main

const configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: fooConfigMap
  namespace: bar
  labels:
    lbl1: somevalue
    lbl2: another
data:
  item1: wow
  item2: wee
`

const configMapHCL = `resource "kubernetes_config_map" "foo_config_map" {
  metadata {
    name      = "fooConfigMap"
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

const basicDeploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: baz-app
  namespace: bat
  annotations:
    foo: fam
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      annotations:
        foo: fam
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        args: ["--debug", "--test"]
        ports:
        - port: 80
          containerPort: 80
        resources:
          requests:
            cpu: "1"
            memory: "512Mi"
          limits:
            cpu: "1"
            memory: "1Gi"
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

const podNodeExporterFullHCL = `resource "kubernetes_pod" "node_exporter_7fth_7" {
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
      termination_message_path   = "/dev/termination-log"
      termination_message_policy = "File"
      image_pull_policy          = "Always"
      security_context {
        privileged = true
      }
    }
    restart_policy                   = "Always"
    termination_grace_period_seconds = 30
    dns_policy                       = "ClusterFirst"
    service_account_name             = "default"
    service_account                  = "default"
    automount_service_account_token  = true
    node_name                        = "gke-cloudlogs-dev-default-pool-4a2a9dae-9b01"
    host_pid                         = true
    scheduler_name                   = "default-scheduler"
    toleration {
      key      = "node.kubernetes.io/not-ready"
      operator = "Exists"
      effect   = "NoExecute"
    }
    toleration {
      key      = "node.kubernetes.io/unreachable"
      operator = "Exists"
      effect   = "NoExecute"
    }
    toleration {
      key      = "node.kubernetes.io/disk-pressure"
      operator = "Exists"
      effect   = "NoSchedule"
    }
    toleration {
      key      = "node.kubernetes.io/memory-pressure"
      operator = "Exists"
      effect   = "NoSchedule"
    }
  }
}
`

const podNodeExporterFullYAML = `apiVersion: v1
kind: Pod
metadata:
  annotations:
    prometheus.io/port: "9100"
    prometheus.io/scheme: http
    prometheus.io/scrape: "true"
  creationTimestamp: 2018-09-10T23:29:40Z
  generateName: node-exporter-
  labels:
    controller-revision-hash: "2418008739"
    name: node-exporter
    pod-template-generation: "1"
  name: node-exporter-7fth7
  namespace: prometheus
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: DaemonSet
    name: node-exporter
    uid: 08e57633-594e-11e8-bdee-42010a800279
  resourceVersion: "42836226"
  selfLink: /api/v1/namespaces/prometheus/pods/node-exporter-7fth7
  uid: 63a4e2b5-b551-11e8-ab49-42010a8000db
spec:
  automountServiceAccountToken: true
  containers:
  - image: prom/node-exporter
    imagePullPolicy: Always
    name: prom-node-exporter
    ports:
    - containerPort: 9100
      name: metrics
      protocol: TCP
    resources: {}
    securityContext:
      privileged: true
      readOnlyRootFilesystem: false
      runAsNonRoot: false
      runAsUser: 0
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-rkd4g
      readOnly: true
  dnsPolicy: ClusterFirst
  hostPID: true
  nodeName: gke-cloudlogs-dev-default-pool-4a2a9dae-9b01
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
  - effect: NoSchedule
    key: node.kubernetes.io/disk-pressure
    operator: Exists
  - effect: NoSchedule
    key: node.kubernetes.io/memory-pressure
    operator: Exists
  volumes:
  - name: default-token-rkd4g
    secret:
      defaultMode: 420
      secretName: default-token-rkd4g
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: 2018-09-10T23:29:43Z
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: 2018-09-10T23:30:21Z
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: 2018-09-10T23:29:43Z
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://320ffc1ab868431b0d51b99d2aac02122b4b03d8274b2c59640e00c7fb6b7a45
    image: prom/node-exporter:latest
    imageID: docker-pullable://prom/node-exporter@sha256:55302581333c43d540db0e144cf9e7735423117a733cdec27716d87254221086
    lastState: {}
    name: prom-node-exporter
    ready: true
    restartCount: 0
    state:
      running:
        startedAt: 2018-09-10T23:30:21Z
  hostIP: 172.16.0.3
  phase: Running
  podIP: 10.8.1.3
  qosClass: BestEffort
  startTime: 2018-09-10T23:29:43Z`

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

const serviceYAML = `apiVersion: v1
kind: Service
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  ports:
  - port: 80
    name: web
  clusterIP: None
  selector:
    app: nginx`

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
const statefulSetYAML = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
  labels:
    app: nginx
spec:
  serviceName: "nginx"
  selector:
    matchLabels:
      app: nginx
  replicas: 14
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: k8s.gcr.io/nginx-slim:0.8
          ports:
            - containerPort: 80
              name: web
          volumeMounts:
            - name: www
              mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
    - metadata:
        name: www
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 1Gi
        storageClassName: thin-disk`
