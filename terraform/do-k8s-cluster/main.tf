terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

variable "do_token" {
}


resource "digitalocean_kubernetes_cluster" "kubernetes_cluster" {
  name    = "my-k8s-cluster"
  region  = "nyc3"
  version = "1.29"

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}


resource "local_file" "kubeconfig" {
  content  = digitalocean_kubernetes_cluster.kubernetes_cluster.kube_config[0].raw_config
  filename = "${path.module}/kubeconfig"
}
