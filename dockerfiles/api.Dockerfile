# Build Stage 

# use ubuntu:focal if need more features in future
FROM ubuntu:focal AS Base 

WORKDIR /app

# Install base Python deps
RUN apt-get update \
    && apt-get install -y \
    --update python3 py3-pip 

# awscli and sam from pip
RUN python3 -m pip install --upgrade pip

# RUN aws --version







