# Build Stage 

# use ubuntu:focal if need more features in future
FROM ubuntu:focal AS Base 

WORKDIR /app

# Install base Python deps
RUN apt-get update \
    && apt-get install -y python3.9 gcc 

RUN apt-get install -y python3-distutils python3-pip python3-apt

RUN pip3 install awscli aws-sam-cli==1.99.0 \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 --version
# awscli and sam from pip
# RUN pip install pip 
    # && pip install awscli aws-sam-cli=1.99.0 \ 
    # && rm -rf /var/lib/apt/lists/*

# CMD [ "make", "_run"]

