# Kubernetes Cheat Sheet

Personal study notes from hands-on labs (kubeadm, CKA-style scenarios): cluster setup, workloads, scheduling, autoscaling, etcd backup/restore, security and kubeconfig.

> **Tip:** Many lab terminals have `alias k=kubectl` set, so `k` and `kubectl` are used interchangeably below.

## Table of Contents

1. [Cluster Setup with kubeadm](#1-cluster-setup-with-kubeadm)
2. [Pods](#2-pods)
3. [ReplicaSets and Deployments](#3-replicasets-and-deployments)
4. [Services](#4-services)
5. [Namespaces](#5-namespaces)
6. [Labels, Selectors and Useful One-Liners](#6-labels-selectors-and-useful-one-liners)
7. [Scheduling](#7-scheduling)
8. [Declarative Management (`kubectl apply`)](#8-declarative-management-kubectl-apply)
9. [Static Pods and Admission Webhooks](#9-static-pods-and-admission-webhooks)
10. [Monitoring, Logs and Pod Config](#10-monitoring-logs-and-pod-config)
11. [Autoscaling (HPA and VPA)](#11-autoscaling-hpa-and-vpa)
12. [Node Maintenance and Cluster Upgrade](#12-node-maintenance-and-cluster-upgrade)
13. [etcd Backup and Restore](#13-etcd-backup-and-restore)
14. [Priority Classes](#14-priority-classes)
15. [Joining Nodes](#15-joining-nodes)
16. [Security: Authentication and TLS](#16-security-authentication-and-tls)
17. [Certificates API (CSR)](#17-certificates-api-csr)
18. [kubeconfig](#18-kubeconfig)
19. [Terminology Quick Reference](#19-terminology-quick-reference)

---

## 1. Cluster Setup with kubeadm

### Overview

1. **Provision** systems or virtual machines.
2. **Designate roles**: one node as control plane (master), the rest as workers.
3. **Install a container runtime** on each node (e.g. containerd).
4. **Install kubeadm** on every node. It bootstraps the cluster by installing and configuring components in the correct order.
5. **Initialize the control plane** with `kubeadm init`.
6. **Configure the pod network** *before* joining workers. A networking add-on is needed for reliable communication between nodes.
7. **Join worker nodes** to the control plane.
8. **Deploy applications** once all nodes are connected.

### Prerequisites (run on **both** nodes)

Set `net.bridge.bridge-nf-call-iptables` to `1`:

```bash
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
br_netfilter
EOF

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF

sudo sysctl --system
```

> The container runtime was pre-installed in the lab, so that step was skipped.

### Install kubeadm, kubelet and kubectl (all nodes)

Lab task: install `kubeadm` and `kubelet` on `controlplane` and `node01` using the exact version `1.32.0-1.1`.

```bash
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.32/deb/Release.key \
  | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.32/deb/ /' \
  | sudo tee /etc/apt/sources.list.d/kubernetes.list

sudo apt-get update

# See available version labels
sudo apt-cache madison kubeadm

sudo apt-get install -y kubelet=1.32.0-1.1 kubeadm=1.32.0-1.1 kubectl=1.32.0-1.1
sudo apt-mark hold kubelet kubeadm kubectl
```

Check the kubelet version:

```bash
kubelet --version
```

### Initialize the control plane

Options used in the lab:

| Option | Value |
|--------|-------|
| `--apiserver-advertise-address` | IP of `eth0` on the controlplane |
| `--apiserver-cert-extra-sans` | `controlplane` |
| `--pod-network-cidr` | `172.17.0.0/16` |
| `--service-cidr` | `172.20.0.0/16` |

```bash
IP_ADDR=$(ip addr show eth0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}')

kubeadm init \
  --apiserver-cert-extra-sans=controlplane \
  --apiserver-advertise-address $IP_ADDR \
  --pod-network-cidr=172.17.0.0/16 \
  --service-cidr=172.20.0.0/16
```

On success, the output ends with:

```text
Your Kubernetes control-plane has initialized successfully!
...
You should now deploy a pod network to the cluster.
...
kubeadm join 192.168.114.75:6443 --token <token> \
        --discovery-token-ca-cert-hash sha256:<hash>
```

Set up the kubeconfig for a regular user:

```bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

Or, as root:

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf
```

> Deploy a pod network add-on next: <https://kubernetes.io/docs/concepts/cluster-administration/addons/>

### Misc

```bash
cat /etc/*release*        # Which Linux version am I on?
```

---

## 2. Pods

```bash
kubectl get pods
kubectl get pods -o wide                  # Node details for pods (quicker than describe)
kubectl describe pod <pod-name>

# Declarative
kubectl create -f pod-definition.yaml

# Imperative
kubectl run mynginx --image=nginx

# Count containers in a pod
kubectl describe pod <pod-name> | grep -c "Containers:"

# Replace an existing pod
kubectl replace -f pod-definition.yaml
kubectl replace --force -f pod-definition.yaml    # Delete and recreate

# Delete all pods in the default namespace
kubectl delete pods --all -n default
```

### Generate a manifest without creating the resource

`--dry-run=client` prevents creation; `-o yaml` prints the manifest.

```bash
kubectl run redis --image=redis --dry-run=client -o yaml > redis-pod.yaml
```

### Troubleshooting a pod (e.g. wrong image name)

After fixing the image name, redeploy:

```bash
kubectl apply -f podCreate.yaml
# or
kubectl edit pod <pod-name>
```

Then check the pod details; `STATUS` should be `Running`.

---

## 3. ReplicaSets and Deployments

### Replication Controller (legacy)

```bash
kubectl create -f rc-definition.yaml
kubectl delete rc myapp-rc
```

### ReplicaSet

```bash
kubectl get replicaset
kubectl replace -f replicaset-definition.yaml        # Declarative: apply changes to existing RS
kubectl scale --replicas=6 -f replicaset-definition.yaml          # Imperative
kubectl scale --replicas=6 replicaset myapp-replicaset            # Imperative (type & name)
kubectl edit replicaset myapp-replicaset                          # Risky: saving applies immediately
kubectl delete replicaset myapp-replicaset
kubectl get replicaset new-replica-set -o yaml > replicaset-definition.yaml
```

> **Tip:** To create a ReplicaSet manifest, generate one for a **Deployment**, then change `kind` and delete the `strategy` block.

### Deployment

```bash
kubectl create -f deployment-definition.yaml
kubectl create deployment --image=nginx nginx

# Generate YAML only (don't create)
kubectl create deployment --image=nginx nginx --dry-run=client -o yaml > nginx-deployment.yaml
kubectl create deployment --image=nginx nginx --replicas=4 --dry-run=client -o yaml > nginx-deployment.yaml
kubectl create deployment --replicas=3 --image=httpd:2.4-alpine httpd-frontend \
  --dry-run=client -o yaml > httpd-deployment-definition.yaml

kubectl get deployments
kubectl get all                  # All resources
kubectl get pods,svc             # Just pods and services
kubectl get pods --watch         # Stream status changes
```

### Rollouts

```bash
kubectl apply -f deployment-definition.yaml          # Edit YAML (e.g. image version), then apply to upgrade
kubectl rollout status deployment/myapp-deployment
kubectl rollout history deployment/myapp-deployment
kubectl rollout undo deployment/myapp-deployment

# Imperative image update (troubleshooting only)
kubectl set image deployment/myapp-deployment nginx-container=nginx:1.9.1
```

---

## 4. Services

- Service types: **ClusterIP**, **NodePort**, **LoadBalancer**.
- NodePort range: **30000 to 32767**.
- A service distributes load across the pods whose labels match the service's selector (random algorithm).

```bash
kubectl create -f service-definition.yaml
kubectl get services
minikube service myapp-service --url        # URL of a service on a minikube node
```

### `expose` vs `create service`

**ClusterIP service `redis-service` for pod `redis` on port 6379**

```bash
# Uses the pod's labels as selectors (recommended)
kubectl expose pod redis --port=6379 --name redis-service --dry-run=client -o yaml

# Assumes selector app=redis; cannot set selectors, so edit the file if labels differ
kubectl create service clusterip redis --tcp=6379:6379 --dry-run=client -o yaml
```

**NodePort service for pod `nginx`, port 80 exposed on node port 30080**

```bash
# Uses the pod's labels, but cannot set nodePort (add it manually to the generated file)
kubectl expose pod nginx --type=NodePort --port=80 --name=nginx-service --dry-run=client -o yaml

# Lets you set nodePort, but does not use the pod's labels
kubectl create service nodeport nginx --tcp=80:80 --node-port=30080 --dry-run=client -o yaml
```

> Each approach has a limitation: one cannot take a selector, the other cannot take a node port. Prefer `kubectl expose`, and if you need a specific node port, generate the file and add `nodePort` manually before creating the service.

**Pod and ClusterIP service in one step:**

```bash
kubectl run httpd --image=httpd:alpine --port=80 --expose
```

---

## 5. Namespaces

```bash
# Permanently switch the current context to a namespace
kubectl config set-context $(kubectl config current-context) --namespace=dev

# Pods across all namespaces
kubectl get pods --all-namespaces
```

---

## 6. Labels, Selectors and Useful One-Liners

```bash
kubectl get pods --selector app=App1                          # Filter by label
kubectl get pod --selector env=prod,bu=finance,tier=frontend  # Multiple filters
kubectl get pods --selector bu=finance | wc -l                # Count (includes header line)
kubectl get all --selector env=prod --no-headers | wc -l      # Count without header
```

---

## 7. Scheduling

### Manual scheduling

- If there is no scheduler, set `nodeName` in the pod spec (e.g. `nodeName: node01`).
- Alternatively, create a `Binding` object and POST it to the pod's binding API, mimicking what the real scheduler does.

### Taints and tolerations

Taints repel pods from nodes; tolerations let specific pods ignore a taint.

```bash
kubectl taint nodes <node-name> <key>=<value>:<taint-effect>
kubectl taint nodes node1 app=blue:NoSchedule

# Remove a taint: note the trailing '-'
kubectl taint nodes controlplane node-role.kubernetes.io/control-plane:NoSchedule-
```

| Effect | Behaviour for pods that do not tolerate the taint |
|--------|---------------------------------------------------|
| `NoSchedule` | Pods will not be scheduled on the node |
| `PreferNoSchedule` | System tries to avoid the node, but no guarantee |
| `NoExecute` | No new pods scheduled, and existing non-tolerating pods are **evicted** |

Toleration in the pod spec:

```yaml
spec:
  tolerations:
    - key: "app"
      operator: "Equal"
      value: "blue"
      effect: "NoSchedule"
```

> **Note:** A taint plus toleration does **not** guarantee the tolerated pod lands on the tainted node; it may also run on untainted nodes. It only *restricts* which pods a node accepts. To force pods onto specific nodes, use **nodeSelector** or **node affinity**.

### Node selector

```bash
kubectl label nodes <node-name> <label-key>=<label-value>
kubectl label nodes node01 size=Large
```

Then reference it in the pod spec under `nodeSelector`.

### Node affinity types

| Type | Meaning |
|------|---------|
| `requiredDuringSchedulingIgnoredDuringExecution` | Pod **must** be placed on a matching node; if none exists, it is not scheduled. Use when placement is crucial. |
| `preferredDuringSchedulingIgnoredDuringExecution` | Scheduler tries to match, but places the pod elsewhere if no match. Use when placement is not crucial. |
| `requiredDuringSchedulingRequiredDuringExecution` *(planned)* | As `required`, and running pods are **evicted** if node labels change so they no longer match. |

- **DuringScheduling**: the pod does not yet exist and is being created for the first time.
- **DuringExecution**: the pod is already running and node labels change afterwards.
- **IgnoredDuringExecution**: already-running pods are not affected by label changes.

---

## 8. Declarative Management (`kubectl apply`)

`kubectl apply` compares **three** sources before deciding what to change:

1. The **local config file** (on your machine).
2. The **live object configuration** (in the cluster, including status fields).
3. The **last applied configuration** (stored as an annotation on the live object).

How it works:

- If the object doesn't exist, it is created.
- On apply, the local YAML is converted to JSON and stored as the *last applied configuration*.
- On later upgrades, all three are compared. If a field was removed from the local file, the last applied configuration lets Kubernetes detect that and remove it from the live object.

> Only `kubectl apply` (declarative) stores the last applied configuration. `kubectl create` and `kubectl replace` do **not**.

---

## 9. Static Pods and Admission Webhooks

### Static pods

- Static pod names have the **node name appended**.
- Another way to identify one: check `ownerReferences` in its YAML. If the owner is a **ReplicaSet**, it is not static; if the owner is a **Node**, it is.

**Find the static pod directory:**

```bash
ps -aux | grep kubelet                              # Find --config=/var/lib/kubelet/config.yaml
grep staticPodPath /var/lib/kubelet/config.yaml
# staticPodPath: /etc/kubernetes/manifests
```

### Admission plugins

```bash
# kube-apiserver runs as a pod, so inspect the process
ps -ef | grep kube-apiserver | grep admission-plugins

# Or via the pod
kubectl get pods -n kube-system
kubectl exec -it kube-apiserver-controlplane -n kube-system -- kube-apiserver -h | grep 'enable-admission-plugins'
```

### Webhook demo

Create the TLS secret for the webhook server in the `webhook-demo` namespace:

```bash
kubectl -n webhook-demo create secret tls webhook-server-tls \
  --cert "/root/keys/webhook-server-tls.crt" \
  --key  "/root/keys/webhook-server-tls.key"
```

Check a pod's security context:

```bash
k get pod pod-with-defaults -o yaml | grep -A2 "securityContext"
#   securityContext:
#     runAsNonRoot: true
#     runAsUser: 1234
```

A pod with a **conflicting** securityContext (must be non-root, but requests UID 0). Without the webhook it would get stuck in `CreateContainerConfigError`; with the webhook, creation is rejected outright.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-with-conflict
  labels:
    app: pod-with-conflict
spec:
  restartPolicy: OnFailure
  securityContext:
    runAsNonRoot: true
    runAsUser: 0
  containers:
    - name: busybox
      image: busybox
      command: ["sh", "-c", "echo I am running as user $(id -u)"]
```

```text
$ k create -f pod-with-conflict.yaml
Error from server: error when creating "pod-with-conflict.yaml": admission webhook
"webhook-server.webhook-demo.svc" denied the request: runAsNonRoot specified, but
runAsUser set to 0 (the root user)
```

---

## 10. Monitoring, Logs and Pod Config

### Metrics server

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

k top nodes                                        # Node resource usage
k top pods                                         # Pod resource usage
k top pods -n default --sort-by='memory'
k top pods -n default --sort-by='cpu' | tail -1    # Least CPU-consuming pod
```

### Logs

```bash
k logs webapp-2
kubectl logs <pod-name> -c <container-name>        # A specific container in a pod
```

### Command and args

Pod running `sleep 5000` in an ubuntu container:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ubuntu-sleeper-2
spec:
  containers:
  - name: ubuntu
    image: ubuntu
    command:
      - "sleep"
      - "5000"
```

`command` overrides the image's ENTRYPOINT and `args` overrides CMD:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ubuntu-sleeper-pod
spec:
  containers:
    - name: ubuntu-sleeper
      image: ubuntu-sleeper
      command: ["sleep2.0"]
      args: ["10"]
```

### Secrets

**Imperative:**

```bash
kubectl create secret generic <secret-name> \
  --from-literal=<key1>=<value1> \
  --from-literal=<key2>=<value2>

kubectl create secret generic app-secret --from-file=app_secret.properties
```

**Declarative:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-ssh-auth
data:
  ssh-privatekey: UG91cmluZzYlRW1vdGljb24lU2N1YmE=
```

```bash
kubectl create -f secretSshauth.yaml
kubectl get secret secret-ssh-auth -o yaml

# Encode / decode
echo -n "Pouring6%Emoticon%Scuba" | base64
echo -n "Pouring6%Emoticon%Scuba" | base64 --decode
```

> Base64 is **encoding, not encryption**. Anyone with access can decode it.

---

## 11. Autoscaling (HPA and VPA)

### Horizontal Pod Autoscaler (HPA)

```yaml
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: nginx-deployment
spec:
  maxReplicas: 3
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
  targetCPUUtilizationPercentage: 80
```

```bash
kubectl apply -f /root/autoscale.yml                                          # Declarative
kubectl autoscale deployment nginx-deployment --max=3 --cpu-percent=80       # Imperative
```

### Vertical Pod Autoscaler (VPA)

**Option A: pre-prepared manifests**

```bash
# 1. Install the VPA CRDs (lets Kubernetes recognise VPA custom resources)
kubectl apply -f /root/vpa-crds.yml

# 2. Install VPA RBAC (permissions to operate in the cluster)
kubectl apply -f /root/vpa-rbac.yml
```

**Option B: clone the upstream repo and run the setup script**

```bash
cd /root
git clone https://github.com/kubernetes/autoscaler.git
cd autoscaler/vertical-pod-autoscaler
./hack/vpa-up.sh
```

**Verify:**

```bash
kubectl get crd                                            # All CRDs
kubectl get crds | grep verticalpodautoscaler              # VPA CRDs

k get deploy -n kube-system | grep vpa-
# vpa-admission-controller   1/1
# vpa-recommender            1/1
# vpa-updater                1/1

k -n kube-system logs <vpa-updater-pod-name>
```

---

## 12. Node Maintenance and Cluster Upgrade

### Drain, cordon and uncordon

```bash
kubectl drain node01        # Evict pods (recreated elsewhere) and mark node unschedulable
kubectl uncordon node01     # Make the node schedulable again
kubectl cordon node02       # Mark unschedulable WITHOUT evicting existing pods
```

### Upgrade with kubeadm (example: v1.31 to v1.32)

Always upgrade the **control plane first**, and drain it before upgrading.

```bash
# 1. Point the apt repo at the next minor release (v1.32)
vim /etc/apt/sources.list.d/kubernetes.list
# deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.32/deb/ /

# 2. Find the available package version
apt update
apt-cache madison kubeadm            # e.g. 1.32.0-1.1

# 3. Upgrade kubeadm, then the cluster
apt-get install kubeadm=1.32.0-1.1
kubeadm upgrade plan v1.32.0         # Also shows latest component versions
kubeadm upgrade apply v1.32.0        # Can take a few minutes

# 4. Upgrade kubelet and restart it
apt-get install kubelet=1.32.0-1.1
systemctl daemon-reload
systemctl restart kubelet
```

Then mark the node schedulable again (`kubectl uncordon controlplane`). Repeat for worker nodes, using `ssh node01` from the controlplane to log in.

Reference: <https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-upgrade/>

---

## 13. etcd Backup and Restore

### Basics

`etcdctl` is the command line client for etcd. In the labs, etcd runs as a **static pod** on the master, using API **v3**.

```bash
export ETCDCTL_API=3
etcdctl snapshot save -h        # See options and mandatory global flags
```

etcd is TLS-enabled, so these flags are mandatory:

| Flag | Purpose |
|------|---------|
| `--cacert` | CA bundle to verify the TLS server |
| `--cert` | Client TLS certificate |
| `--key` | Client TLS key |
| `--endpoints` | Default `127.0.0.1:2379` (etcd on the master, exposed on localhost) |

**Find the etcd address** (look for `--listen-client-urls`):

```bash
kubectl describe pod etcd-controlplane -n kube-system
k -n kube-system describe pod etcd-controlplane | grep '\-\-'     # Show all flags incl. cert paths
```

### Back up (stacked etcd)

Use case: the master is being rebooted for maintenance, so take a snapshot first.

```bash
ETCDCTL_API=3 etcdctl --endpoints=https://[127.0.0.1]:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot save /opt/snapshot-pre-boot.db
```

### Restore (stacked etcd)

```bash
ETCDCTL_API=3 etcdctl --data-dir /var/lib/etcd-from-backup \
  snapshot restore /opt/snapshot-pre-boot.db
```

Restoring to a different directory on the same server needs only `--data-dir`. Then edit `/etc/kubernetes/manifests/etcd.yaml` and change the `hostPath` of the `etcd-data` volume:

```yaml
volumes:
- hostPath:
    path: /var/lib/etcd-from-backup      # was /var/lib/etcd
    type: DirectoryOrCreate
  name: etcd-data
```

Notes:

- The etcd pod is static, so it is recreated automatically when the manifest changes. kube-controller-manager and kube-scheduler restart too; wait 1 to 2 minutes (`watch "crictl ps | grep etcd"`).
- If etcd doesn't reach `1/1 Ready`, run `kubectl delete pod -n kube-system etcd-controlplane` and wait a minute.
- Optionally also update `--data-dir` and the `volumeMounts` `mountPath` in the manifest. This is **not required** for the restore.

### Working with multiple clusters

```bash
k config get-clusters                       # How many clusters?
k config use-context cluster1               # Switch cluster
k config view                               # List clusters and contexts
```

Find the etcd data directory and server address:

```bash
ps -ef | grep --color=always 'etcd.*--data-dir'                 # On the etcd host
k -n kube-system describe pod kube-apiserver-<cluster>-controlplane   # Look at --etcd-servers=
```

### External etcd

If `kubectl get pods -n kube-system | grep etcd` returns nothing **and** `/etc/kubernetes/manifests/` has no etcd manifest, etcd is external. Confirm via the API server:

```bash
ps -ef | grep etcd
# kube-apiserver ... --etcd-servers=https://192.160.244.3:2379 ...
```

There is no local etcd at `127.0.0.1` in this case. SSH to the etcd server to work with it.

**How many members in the etcd cluster?**

```bash
ETCDCTL_API=3 etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/etcd/pki/ca.pem \
  --cert=/etc/etcd/pki/etcd.pem \
  --key=/etc/etcd/pki/etcd-key.pem \
  member list
# 59ee55985632d394, started, etcd-server, https://192.160.244.3:2380, https://192.160.244.3:2379, false
```

One line of output means a single-member cluster.

**Copy a backup between machines:**

```bash
scp cluster1-controlplane:/opt/cluster1.db /opt/
```

### Restore (external etcd)

Scenario: restore `/opt/cluster2.db` to `/var/lib/etcd-data-new` on cluster2, and make sure the objects in the `critical` namespace come back.

```bash
kubectl config use-context cluster2
```

**Step 1.** Copy the snapshot to the etcd server:

```bash
scp /opt/cluster2.db etcd-server:/root
```

**Step 2.** Restore on the etcd server (using the same certs found earlier):

```bash
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/etcd/pki/ca.pem \
  --cert=/etc/etcd/pki/etcd.pem \
  --key=/etc/etcd/pki/etcd-key.pem \
  snapshot restore /root/cluster2.db --data-dir /var/lib/etcd-data-new
```

**Step 3.** Update `--data-dir` in the systemd unit (`vi /etc/systemd/system/etcd.service`):

```ini
[Unit]
Description=etcd key-value store
Documentation=https://github.com/etcd-io/etcd
After=network.target

[Service]
User=etcd
Type=notify
ExecStart=/usr/local/bin/etcd \
  --name etcd-server \
  --data-dir=/var/lib/etcd-data-new \
  ...
```

**Step 4.** Fix ownership (must be the `etcd` user):

```bash
chown -R etcd:etcd /var/lib/etcd-data-new
ls -ld /var/lib/etcd-data-new/        # drwx------ 3 etcd etcd ...
```

**Step 5.** Reload and restart:

```bash
systemctl daemon-reload
systemctl restart etcd
```

**Step 6 (optional, recommended).** Restart control plane components (kube-scheduler, kube-controller-manager, kubelet) so they don't rely on stale data.

---

## 14. Priority Classes

```bash
kubectl describe priorityclass system-node-critical
kubectl get priorityclasses
```

**Create `high-priority`** (value 100000, `PreemptLowerPriority`, not the global default):

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 100000
globalDefault: false
description: "This priority class is used for high-priority pods."
preemptionPolicy: PreemptLowerPriority
```

```bash
kubectl apply -f high-priority.yaml
```

**Pod using the class:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: high-prio-pod
  name: high-prio-pod
spec:
  containers:
  - image: nginx
    name: high-prio-pod
  priorityClassName: high-priority
```

**Compare priority classes across pods:**

```bash
kubectl get pods -o custom-columns="NAME:.metadata.name,PRIORITY:.spec.priorityClassName"
```

### Fixing a pod stuck without priority (`critical-app`)

Assign `high-priority`, delete and recreate the pod, and confirm it is running.

```bash
kubectl get pod critical-app -o yaml > critical-app.yaml
```

Edit the saved copy:

```yaml
spec:
  containers:
  - image: nginx
    name: critical-container
  priorityClassName: high-priority      # Add this
  preemptionPolicy: PreemptLowerPriority
  # priority: 0                         # Remove this line (old default priority)
```

```bash
kubectl delete pod critical-app
kubectl apply -f critical-app.yaml
kubectl get pod critical-app            # Should be Running
```

---

## 15. Joining Nodes

Generate a fresh join command on the controlplane (or reuse the one printed by `kubeadm init`):

```bash
kubeadm token create --print-join-command
```

Then SSH to the worker and run it:

```bash
ssh node01
kubeadm join 192.168.114.75:6443 --token <token> \
  --discovery-token-ca-cert-hash sha256:<hash>
```

Expected result:

```text
This node has joined the cluster:
* Certificate signing request was sent to apiserver and a response was received.
* The Kubelet was informed of the new secure connection details.

Run 'kubectl get nodes' on the control-plane to see this node join the cluster.
```

---

## 16. Security: Authentication and TLS

### Authentication methods for the kube-apiserver

1. **Static password file** (CSV: password, username, user ID, optional group)
2. **Static token file**
3. **TLS client certificates**
4. **Identity services** (LDAP, Kerberos, etc.)

**Static password file.** Pass the file to kube-apiserver via `--basic-auth-file` (in the `ExecStart` of `kube-apiserver.service`, or under `command` in the pod definition if set up with kubeadm).

```bash
curl -v -k https://master-node-ip:6443/api/v1/pods -u "username1:Password1"
```

**Static token file.** Pass it via `--token-auth-file`.

```bash
curl -v -k https://master-node-ip:6443/api/v1/pods \
  --header "Authorization: Bearer <token>"
```

> Methods 1 and 2 are **insecure and not recommended** (and basic auth has been removed from recent Kubernetes versions). With kubeadm, use volume mounts to pass the auth file.

### Who needs which certificates

**Servers (server certificates):**

- kube-apiserver
- etcd
- kubelet (HTTPS API, for communication with worker nodes)

**Clients (talk to the kube-apiserver):**

| Client | Certificate / key |
|--------|-------------------|
| Admin user | `admin.crt` / `admin.key` |
| kube-scheduler | `scheduler.crt` / `scheduler.key` (picks the right nodes for pods) |
| kube-controller-manager | `controller-manager.crt` / `controller-manager.key` |
| kube-proxy | `kube-proxy.crt` / `kube-proxy.key` |

The **CA** needs root certificates (`ca.crt`, `ca.key`).

### Generating certificates with openssl

```bash
# 1. Private key
openssl genrsa -out ca.key 2048

# 2. Certificate signing request (CN = component name, O = group)
openssl req -new -key ca.key -subj "/CN=KUBERNETES-CA" -out ca.csr
#   e.g. for an admin:  -subj "/CN=kube-admin/O=system:masters"

# 3. Sign the certificate
openssl x509 -req -in admin.csr -CA ca.crt -CAkey ca.key -out admin.crt
```

- A CSR is like a certificate with all your details but **no signature**. `CN` is the component name (this is what you see in audit logs). `/O` assigns the user to a group such as `system:masters`.
- Client certs for system components must be prefixed with `system:` (e.g. `SYSTEM:KUBE-SCHEDULER`, `SYSTEM:KUBE-CONTROLLER-MANAGER`). Create the rest the same way.
- The **CA root certificate must be copied to all components**.
- For the kube-apiserver, list all alias names under `[alt_names]` in `openssl.cnf`, then use `openssl req ... -config openssl.cnf`.
- etcd and kubelet need similar CA, key and server certs. Kubelets need one per node (`node01`, `node02`, ...).
- Flags: `--client-ca-file` for `ca.crt`, `--tls-cert-file` for the API server cert.

### Using the certificates

```bash
curl https://master-node-ip:6443/api/v1/pods \
  --key admin.key --cert admin.crt --cacert ca.crt
```

Or put the settings in a **kubeconfig** (the method most clients use).

### Where certificates live

| Setup | Location |
|-------|----------|
| Manual | `/etc/systemd/system/kube-apiserver.service` |
| kubeadm | `/etc/kubernetes/manifests/kube-apiserver.yaml` |

**Decode / inspect a certificate:**

```bash
openssl x509 -in /etc/kubernetes/pki/apiserver.crt -text -noout
```

### Inspecting logs

```bash
# Manual setup
journalctl -u etcd.service -l

# kubeadm setup
kubectl logs etcd-master

# If kube-apiserver or etcd is down
docker ps -a
docker logs <container-id>
```

### Troubleshooting: kube-apiserver keeps exiting

```bash
crictl ps -a | grep kube-apiserver
crictl logs --tail=2 <container-id>
```

Example error:

```text
transport: authentication handshake failed: x509: certificate signed by unknown authority
```

This indicates the etcd CA certificate configured for the API server is wrong. In `/etc/kubernetes/manifests/kube-apiserver.yaml`, set the etcd CA to **`/etc/kubernetes/pki/etcd/ca.crt`**. Save, then wait a couple of minutes for the pod to become Ready.

---

## 17. Certificates API (CSR)

Kubernetes has its own API for certificate signing requests:

1. Create a `CertificateSigningRequest` object.
2. Review the requests.
3. Approve them with `kubectl`.
4. Extract the certificate and send it to the user.

The component doing the work is the **controller manager** (CSR-APPROVING and CSR-SIGNING controllers).

### Workflow

```bash
# 1. User generates a key
openssl genrsa -out Djiadmin.key 2048

# 2. User creates a CSR and sends it to the admin
openssl req -new -key Djiadmin.key -subj "/CN=Dji" -out Djicsr.csr

# 3. Admin creates a CertificateSigningRequest object (see manifest below).
#    The 'request' field holds the CSR, base64-encoded (not plain text):
cat Djicsr.csr | base64 -w 0        # -w 0 prints without line wraps

# 4. List requests
kubectl get csr

# 5. Approve
kubectl certificate approve Dji

# 6. View the approved certificate (base64 in .status.certificate)
kubectl get csr Dji -o yaml

# 7. Decode it to plain text and share with the user
echo "<base64-cert>" | base64 --decode

# Delete a CSR
kubectl delete csr agent-smithcsr
```

### Example manifest (`akshay`)

From Kubernetes 1.19 the API is `certificates.k8s.io/v1`, which **requires `signerName`**. For client authentication to the API server use `kubernetes.io/kube-apiserver-client`.

```yaml
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: akshay
spec:
  groups:
  - system:authenticated
  request: <base64-encoded contents of akshay.csr>
  signerName: kubernetes.io/kube-apiserver-client
  usages:
  - client auth
```

```bash
kubectl apply -f akshay-csr.yaml
```

---

## 18. kubeconfig

### Calling the API manually

```bash
# curl
curl https://my-kubernetes-hostname:6443/api/v1/pods \
  --key admin.key --cert admin.crt --cacert ca.crt

# kubectl with explicit flags
kubectl get pods \
  --server my-kubernetes-hostname:6443 \
  --client-key admin.key \
  --client-certificate admin.crt \
  --certificate-authority ca.crt
```

Sample response:

```json
{
  "kind": "PodList",
  "apiVersion": "v1",
  "metadata": { "selfLink": "/api/v1/pods" },
  "items": []
}
```

### Using a kubeconfig file

Put those same settings in a `config` file, then:

```bash
kubectl get pods --kubeconfig config
```

By default, kubectl reads **`$HOME/.kube/config`**, so no `--kubeconfig` flag is needed if the file is saved there.

### Sections of a kubeconfig

| Section | Purpose |
|---------|---------|
| **Clusters** | The Kubernetes clusters you need access to |
| **Users** | Existing user accounts with access to clusters (you are not creating new users here) |
| **Contexts** | Tie a user to a cluster, defining who accesses which cluster |

---

## 19. Terminology Quick Reference

| Term | Meaning |
|------|---------|
| **Replication Controller** | Older way to run multiple pod replicas for load balancing; obsolete |
| **ReplicaSet** | Newer replacement (`apiVersion: apps/v1`, `selector` is mandatory) |
| **Pod** | `apiVersion: v1` |
| **Rolling upgrade** | Gradually replacing old pods with new ones |
| **Cilium / Flannel** | Pod network add-ons for internal routing and IP allocation between nodes and pods |
| **ClusterIP** | Internal-only service |
| **NodePort** | Exposes a service on each node (30000 to 32767) |
| **LoadBalancer** | Exposes a service via an external load balancer |
