# Digital Ocean Kubernetes Cluster

Creates a 3 node(2 vCPU, 2GB Memory) kubernetes cluster(v1.29)  in `nyc3` region

```bash
export TF_VAR_do_token=your_digitalocean_token
```
```bash 

terraform init
terraform plan
terraform apply -auto-approve
```
