docker pull hyperledger/fabric-orderer:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-orderer:latest
docker tag hyperledger/fabric-orderer:x86_64-1.0.0-beta hyperledger/fabric-orderer:latest
docker pull hyperledger/fabric-peer:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-peer:latest 
docker tag hyperledger/fabric-peer:x86_64-1.0.0-beta hyperledger/fabric-peer:latest
docker pull hyperledger/fabric-ca:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-ca:latest
docker tag hyperledger/fabric-ca:x86_64-1.0.0-beta hyperledger/fabric-ca:latest
docker pull hyperledger/fabric-kafka:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-kafka:latest
docker tag hyperledger/fabric-kafka:x86_64-1.0.0-beta hyperledger/fabric-kafka:latest
docker pull hyperledger/fabric-zookeeper:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-zookeeper:latest
docker tag hyperledger/fabric-zookeeper:x86_64-1.0.0-beta hyperledger/fabric-zookeeper:latest
docker pull hyperledger/fabric-ccenv:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-ccenv:latest 
docker tag hyperledger/fabric-ccenv:x86_64-1.0.0-beta hyperledger/fabric-ccenv:latest
docker pull hyperledger/fabric-couchdb:x86_64-1.0.0-beta
docker rmi hyperledger/fabric-couchdb:latest 
docker tag hyperledger/fabric-couchdb:x86_64-1.0.0-beta hyperledger/fabric-couchdb:latest
docker rmi hyperledger/fabric-baseos:x86_64-0.3.1 
docker pull hyperledger/fabric-baseos:x86_64-0.3.1 
docker tag hyperledger/fabric-baseos:x86_64-0.3.1 hyperledger/fabric-baseos:latest
docker rmi hyperledger/fabric-baseimage:x86_64-0.3.1 
docker pull hyperledger/fabric-baseimage:x86_64-0.3.1 
docker tag hyperledger/fabric-baseimage:x86_64-0.3.1 hyperledger/fabric-baseimage:latest

#https://registry-1.docker.io/v2/hyperledger/fabric-baseos

#docker rmi fabric-baseimage:latest




