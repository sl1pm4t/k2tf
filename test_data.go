package main

const ycm = `
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

const yd1 = `
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

const y = `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: baz-app
  namespace: bat
  annotations:
    foo: fam
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        foo: fam
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - port: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bazApp
  namespace: bat
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        foo: fam
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - port: 80
---
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
