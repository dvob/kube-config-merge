# kube-config-merge
Merges Kubernetes configuration from SOURCE into the current configuration. It uses the default kubectl config locations or you can explicitly specify a target config using `--kubeconfig` flag.
If no SOURCE is specified it is read from standard input.

```
kube-config-merge some-kubeconfig.yaml

kube-config-merge --kubeconfig my-target-config.yaml some-kubeconfig.yaml

ssh k3s-host sudo cat /etc/rancher/k3s/k3s.yaml | kube-config-merge --name k3s --server https://k3s-host

ssh kubeadm-host sudo cat /etc/kubernetes/admin.conf | kube-config-merge --name kubeadm --server https://kubeadm-host
```

# Install
```
go install github.com/dvob/kube-config-merge@latest
```
