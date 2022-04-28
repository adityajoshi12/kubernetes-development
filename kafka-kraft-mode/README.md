# Kafka without zookeeper

### Overview
Apache Kafka is a distributed streaming platform that is the foundation for many event-driven systems. It allows for applications to produce and consume events on various topics with built-in fault tolerance.

Prior to v2.8 of Kafka, all Kafka instances required Zookeeper to function. Zookeeper has been used as the metadata storage for Kafka, providing a way to manage brokers, partitions, and tasks such as providing consensus when electing the controller across brokers.

Apache Kafka v2.8 now has experimental support for running without Zookeeper: Kafka Raft Metadata mode (KRaft mode). KRaft mode was proposed in Kafka Improvement Proposal (KIP) KIP-500. KRaft mode Kafka now runs as a single process and a single distributed system, making it simpler to deploy, operate, and manage Kafka, especially when running Kafka on Kubernetes. KRaft mode also allows Kafka to more easily run with less resources, making it more suitable for edge computing solutions.

### Cluster components
1. Namespace `kafka-kraft`
2. PersistentVolume `kafka-pv-volume`
3. PersistentVolumeClaim `kafka-pv-claim`
4. Service `kafka-svc`
5. StatefulSet `kafka`

### Steps
1. Build the docker image for kraft
```
cd kubernetes/docker
docker build -t adityajoshi12/kafka-kraft:1.0 .
```
2. Push your docker image to image registry like docker hub.
3. Starting Kafka
```
kubectl apply -f kubernetes/kafka.yaml
```
