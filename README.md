# Meet Near Me API

## Getting Started

1. `$ docker compose build`
1. `$ docker compose up`
1.


## Validating User Basic end Points 
User below commands 

`curl localhost:3001/user` - list table Users
`curl -X POST -H 'Content-Type: application/json' -d '{"name": "Brandon", "kind": "user", "region": "USA"}' http://0.0.0.0:3001/user` - insert new user



### Reference for interacting with dynamodb from aws cli v2
https://awscli.amazonaws.com/v2/documentation/api/latest/reference/dynamodb/index.html
