# Meet Near Me API

## Getting Started

### Running the local SAM dynamodb docker container

1. `$ docker compose build`
1. `$ docker compose up`

### Running the Lambda project

1. `npm i`
1. Create an AWS account if you don't have one
1. [Authorize SST via AWS CLI](https://sst.dev/chapters/configure-the-aws-cli.html)
1. `npm run dev` runs the Go Lambda Gateway V2 server locally, proxied through
   Lambda to your local

### Generate Go templates from \*.templ files

1. Add to your `.zshrc` / `.bashrc`

```
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

1. `go install github.com/a-h/templ`
1. Run `templ generate`

### Validate home page is working + data-connected

`npm run dev` should finish with an AWS endpoint, hitting that endpoint should
show a list of events in that particular stage's dynamoDb table

## Validating Event Basic end Points

### API Example Curl Requests

`curl <AWS URL from npm run dev>` - list table Events
`curl -X POST -H 'Content-Type: application/json' -d '{"name": "Chess Tournament", "description": "Join the junior chess tournament to test your abilities", "datetime": "2024-03-13T15:07:00", "address": "15 Chess Street", "zip_code": "84322", "country": "USA"}' <AWS URL from npm run dev>` -
insert new event

### Reference for interacting with dynamodb from aws cli v2

https://awscli.amazonaws.com/v2/documentation/api/latest/reference/dynamodb/index.html
