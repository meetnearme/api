FROM ubuntu:focal

WORKDIR /app

# Install base Python deps
RUN apt-get update \
    && apt-get install -y python3.9 gcc make

RUN apt-get install -y python3-distutils python3-pip python3-apt sudo

# install awscli and sam 
RUN pip3 install awscli aws-sam-cli==1.99.0 \
    && rm -rf /var/lib/apt/lists/*

# install software commons 
RUN sudo apt update \
    && sudo apt install -y software-properties-common

# isntall golang
RUN sudo add-apt-repository ppa:longsleep/golang-backports -y \
    && sudo apt update \ 
    && sudo apt install -y golang-go

ENV GOROOT /usr/lib/go
ENV GOTPATH /go
ENV PATH /go/bin:$PATH

RUN mkdir -p ${GOROOT}/src ${GOTPATH}/bin
    
RUN go version 

# Below is running command to hold open while docker compose works
# CMD [ "sh", "-c", "while sleep 3600; do:; done" ]
