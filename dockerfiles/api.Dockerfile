FROM ubuntu:focal

WORKDIR /app

# Install base Python deps
RUN apt-get update \
    && apt-get install -y python3.9 gcc make

RUN apt-get install -y python3-distutils python3-pip python3-apt sudo

# install awscli and sam 
RUN pip3 install awscli aws-sam-cli==1.99.0 \
    && rm -rf /var/lib/apt/lists/*

# install software commons and golang from apt-get 
RUN sudo apt-get update \ 
    && sudo apt-get install -y golang \
    && sudo apt-get install -y software-properties-common

# install golang backports and update to newest version
RUN sudo add-apt-repository ppa:longsleep/golang-backports -y \
    && sudo apt update \ 
    && sudo apt install -y golang-1.20

ENV GOROOT /usr/lib/go
ENV GOTPATH /go
ENV PATH /go/bin:$PATH

RUN mkdir -p ${GOROOT}/src ${GOTPATH}/bin
    
RUN go version 
