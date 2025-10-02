# Introduction
The IBM Cloud VPC CNI plugin for Kubernetes is responsible for providing seamless networking between pods running in a Kubernetes cluster and the underlying IBM Cloud Virtual Private Cloud (VPC) infrastructure. It enables pods to receive IP addresses directly from the VPC subnet, allowing them to communicate natively within the VPC network without requiring NAT or additional overlays. This approach ensures high-performance networking and simplifies integration with other IBM Cloud VPC services.

By leveraging the IBM VPC CNI, each pod gets an IP address from the same subnet as the worker node, making pod-to-pod, pod-to-service, and pod-to-external communication straightforward and consistent. This model improves network scalability, supports enterprise-grade security policies, and provides better observability compared to traditional overlay-based CNI solutions.

This CNI has been tested in the following environments

| Ubuntu  | Kernel  | Kubernetes | CNI |  
|---------|---------|-----------|-------
| 24.04   | 6.8.0-1033-ibm | 1.33.5 | 0.4.0 |

# Preconditions
* Ensure Go 1.23 (or later) is installed.
* This CNI is supported only for VSI workers on VPC in IBM Cloud.
* A Trusted Profile with VPC administrator permissions must be used.
* Instance Metadata Service must be enabled.
* Public internet towards VPC iaas endpoint must be enabled ```${region}.iaas.cloud.ibm.com```

# Current features
* Integration with Trusted Profiles.
* Supports multiple VNIs.
* Manages subnet IP reservation, including assigning, releasing, and binding IPs as secondary IPs on VNIs.
* Manage network veth-pair devices on workers.
* The solution provide 7 × the number of VNI IPs per worker for pods.

# Backlog (next steps)
* Enable private communication only (`${region}.private.iaas.cloud.ibm.com`).
* Test in an environment with few IPs available.
* Improve the way network device names are created to avoid collisions 100%. 
* Primary IPs for non-primary VNIs are currently not being used.
* Allow apikey integration.
* Enable Auto-release option when IP is bound. 

# How to use it
But before.... If you need a Kubernetes cluster infrastructure go [here](#provision-kubernetes-infrastructure). Remember that you need access to both the master and worker nodes to test it properly.

Skip this option if you have Go installed (on Ubuntu)
```bash
sudo apt install wget tar make -y

wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz

cat << EOF >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
EOF
source ~/.bashrc

go --version
```
Compile it on x86 machine!
```bash
cd ibmcloud-vpc-cni-k8s
make
```
### Manual configuration
Upload it to workers and copy it to ```/opt/cni/bin/``` directory
```bash
sudo cp ibmcloud-vpc-cni-k8s /opt/cni/bin/
sudo chmod +x /opt/cni/bin/ibmcloud-vpc-cni-k8s
```
Create 10-ibm-vpc-cni.conf file
```bash
sudo tee /etc/cni/net.d/10-ibm-vpc-cni.conf > /dev/null <<EOF
{
    "cniVersion": "0.4.0",
    "name": "ibm-vpc-cni",
    "type": "ibmcloud-vpc-cni-k8s",
    "ipam": {
      "type": "static"
    }
}
EOF
```
Restart kubelet
```bash
service kubelet restart
```
Verify workers are ready
```bash
kubectl get nodes
```
Create example pods
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:
    app: nginx
spec:
  containers:
    - name: nginx
      image: nginx:latest
      ports:
        - containerPort: 80
EOF
```
Check if IP has been bound
```bash
kubectl get pod -o wide
```
Delete pod and check the subnet ip has being disassociated
```bash
kubectl delete pod nginx-pod
```

# Utils

### Install Go on Ubuntu
```bash
sudo apt install wget tar make -y

wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz

cat << EOF >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
EOF
source ~/.bashrc

go --version
```

### Provision Kubernetes infrastructure
The intend of this section is to provision the following components. We will keep the solution as simple as possible.
* 1 VPC
* 2 Subnets (```10.1.1.0/24```, ```10.1.2.0/24```)
* 3 VSI (1 master and 2 workers)

Create the RSA key pair required for the virtual server instance that will be created in the next step.
```bash
ssh-keygen -f key -t rsa
```
Create a single terraform main.tf file with the following content to deploy the resources required.
```bash
terraform {
  required_providers {
    ibm = {
      source = "IBM-Cloud/ibm"
      version = "1.83.1"
    }
  }
}

provider "ibm" {
}

data "ibm_is_region" "region" {
}

#IAM
resource "ibm_iam_trusted_profile" "iam_trusted_profile" {
  name = "tp-cni"
}

resource "ibm_iam_trusted_profile_claim_rule" "iam_trusted_profile_claim_rule" {
  profile_id = ibm_iam_trusted_profile.iam_trusted_profile.id
  type       = "Profile-CR"
  conditions {
    claim    = "region"
    operator = "EQUALS"
    value    = "\"${data.ibm_is_region.region.name}\""
  }
  name    = "rule"
  cr_type = "VSI"
}

resource "ibm_iam_trusted_profile_policy" "policy" {
  profile_id  = ibm_iam_trusted_profile.iam_trusted_profile.id
  roles       = ["Manager"]

  resources {
    service = "is"
  }
}

# VPC
resource "ibm_is_vpc" "vpc" {
  name = "vpc-cni"
  address_prefix_management = "manual"
}

resource "ibm_is_vpc_address_prefix" "ibm_is_vpc_address_prefix01" {
  name = "address-vpc-zone1"
  zone = "${data.ibm_is_region.region.name}-1"
  vpc  = ibm_is_vpc.vpc.id
  cidr = "10.1.1.0/24"
}

resource "ibm_is_subnet" "subnet01" {
  name            = "subnet01"
  vpc             = ibm_is_vpc.vpc.id
  zone            = "${data.ibm_is_region.region.name}-1"
  ipv4_cidr_block = "10.1.1.0/24"
  depends_on = [ibm_is_vpc_address_prefix.ibm_is_vpc_address_prefix01]
}

resource "ibm_is_vpc_address_prefix" "ibm_is_vpc_address_prefix02" {
  name = "address-vpc-zone2"
  zone = "${data.ibm_is_region.region.name}-2"
  vpc  = ibm_is_vpc.vpc.id
  cidr = "10.1.2.0/24"
}

resource "ibm_is_subnet" "subnet02" {
  name            = "subnet02"
  vpc             = ibm_is_vpc.vpc.id
  zone            = "${data.ibm_is_region.region.name}-2"
  ipv4_cidr_block = "10.1.2.0/24"
  depends_on = [ibm_is_vpc_address_prefix.ibm_is_vpc_address_prefix02]
}

# SG
resource "ibm_is_security_group" "sg" {
  name = "sg-cni"
  vpc  = ibm_is_vpc.vpc.id
}

resource "ibm_is_security_group_rule" "sg_rule_inbound" {
  group     = ibm_is_security_group.sg.id
  direction = "inbound"
  remote    = "0.0.0.0/0"
}

resource "ibm_is_security_group_rule" "sg_rule_outbound" {
  group     = ibm_is_security_group.sg.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
}

# VSI
data "ibm_is_image" "image"{
    name = "ibm-ubuntu-24-04-3-minimal-amd64-2"
}

resource "ibm_is_ssh_key" "key" {
  name       = "cni-ssh-key"
  public_key = file("key.pub")
  type       = "rsa"
}

locals {
  vsis = [
    {
      name = "master-01"
      zone = ibm_is_subnet.subnet01.zone
    },
    {
      name = "worker-01"
      zone = ibm_is_subnet.subnet01.zone
    },
    {
      name = "worker-02"
      zone = ibm_is_subnet.subnet02.zone
    }
  ]
}

resource "ibm_is_instance" "vsi" {
  for_each = { for v in local.vsis : v.name => v }

  name    = each.value.name
  image   = data.ibm_is_image.image.id
  profile = "cx3d-4x10"

  primary_network_attachment {
    name = "${each.value.name}-attach"
    virtual_network_interface {
      name            = each.value.name
      subnet          = each.value.zone == ibm_is_subnet.subnet01.zone ? ibm_is_subnet.subnet01.id : ibm_is_subnet.subnet02.id
      security_groups = [ibm_is_security_group.sg.id]
    }
  }

  vpc  = ibm_is_vpc.vpc.id
  zone = each.value.zone
  keys = [ibm_is_ssh_key.key.id]
  default_trusted_profile_target = ibm_iam_trusted_profile.iam_trusted_profile.id
  metadata_service {
     enabled = true
     protocol = "https"
  } 
}

resource "ibm_is_floating_ip" "floating" {
  for_each = { for v in local.vsis : v.name => v }

  name = "floatingip-${each.value.name}"
  zone = each.value.zone
}

resource "ibm_is_virtual_network_interface_floating_ip" "vni_floating" {
  for_each = { for v in local.vsis : v.name => v }

  virtual_network_interface = ibm_is_instance.vsi[each.key].primary_network_attachment[0].virtual_network_interface[0].id
  floating_ip              = ibm_is_floating_ip.floating[each.key].id
}

output "floating_addresses" {
  value = {
    for name, v in ibm_is_virtual_network_interface_floating_ip.vni_floating :
    name => v.address
  }
}
```
Set the following environment variables needed to run IBM Cloud Terraform provider.
```bash
export IC_REGION="eu-es"
export IC_API_KEY=PUT_HERE_YOUR_API_KEY
```
Let's provision the infrastructure!. Run the following commands.
```bash
terraform init
terraform plan -out plan.out
terraform apply plan.out
```
Repeat the following steps for every machine. Log into
```bash 
ssh -i key root@PUBLICIP
```
Update system (master and workers)
```bash
sudo apt update && sudo apt upgrade -y
```
Disable swap (master and workers)
```bash
sudo swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab
```
Load some modules needed (master and workers)
```bash
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter
```
Sysctl configuration (master and workers)
```bash
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system
```
Install containerd (master and workers)
```bash
sudo apt install -y containerd

sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml

sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd
```
Check it (master and workers)
```bash
sudo ctr containers list
```
Install kubernetes (master and workers)
```bash
sudo apt install -y apt-transport-https ca-certificates curl gpg

sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.33/deb/Release.key | \
    sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.33/deb/ /" | \
    sudo tee /etc/apt/sources.list.d/kubernetes.list

sudo apt update
sudo apt install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```
Init master (only master)
```bash
sudo kubeadm init --pod-network-cidr=10.1.0.0/16 --cri-socket unix:///run/containerd/containerd.sock
```
Get 2 times the command to join workers (only master)
```bash
kubeadm token create --print-join-command
kubeadm token create --print-join-command
```
For every worker run one of previous command shown on stdout (workers)
```bash
#Just an example
kubeadm join 10.1.1.4:6443 --token asxf91.han7v1778cwuw54g --discovery-token-ca-cert-hash sha256:a2201e3c35dca5ba87fcc4f875d4a73bc6997b335a5f1ccaabff24e7ef5dc433
```
Configure kubectl on master directly (master)
```bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```
Finally on master (master)
```bash
kubectl get nodes -o wide
kubectl get pods -A
```
If you want to test it on a machine other than the master or worker nodes, you must copy the file ```/etc/kubernetes/admin.conf```. From the master to your machine’s kubeconfig location (~/.kube/config). Then, on your machine, run:
```bash
kubectl get nodes
kubectl get pods -A
```
Finally install a CNI. To install calico just simply do. **Skip it if you intend to install IBM Cloud VPC CNI!**
```bash
kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.28.0/manifests/calico.yaml
```


