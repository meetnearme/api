GOOS=linux go build -o main
sam local start-api --skip-pull-image -p 3001 --host=0.0.0.0 --container-host-interface=127.0.0.1 --container-host=host.docker.internal --docker-network aws_backend --warm-containers EAGER
