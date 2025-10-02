# Introduction
The IBM Cloud VPC CNI plugin for Kubernetes is responsible for providing seamless networking between pods running in a Kubernetes cluster and the underlying IBM Cloud Virtual Private Cloud (VPC) infrastructure. It enables pods to receive IP addresses directly from the VPC subnet, allowing them to communicate natively within the VPC network without requiring NAT or additional overlays. This approach ensures high-performance networking and simplifies integration with other IBM Cloud VPC services.

By leveraging the IBM VPC CNI, each pod gets an IP address from the same subnet as the worker node, making pod-to-pod, pod-to-service, and pod-to-external communication straightforward and consistent. This model improves network scalability, supports enterprise-grade security policies, and provides better observability compared to traditional overlay-based CNI solutions.

This CNI has been tested in the following environments

| Ubuntu  | Kernel  | Kubernetes | CNI |  
|---------|---------|-----------|-------
| 24.04   | 6.8.0-1033-ibm | 1.33.5 | 0.4.0 |

# Preconditions
* Have Go 1.23 version
* CNI is only valid for VSI VPC on IBM Cloud
* TP must be bound with VPC admin permissions
* Metadata must be enabled

# How to use it
Compite it
```bash
make
```
### Manual configuration
Upload it to workers
```bash
sudo cp ibmcloud-vpc-cni-k8s /opt/cni/bin/
sudo mkdir -p /opt/cni/bin
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

### Install Go on Centos 9
```bash
sudo dnf install wget tar make -y

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