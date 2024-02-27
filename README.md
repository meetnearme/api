# Meet Near Me API

## Getting Started

### Running the local SAM dynamodb docker container

1. `$ docker compose build`
2. `$ docker compose up`

### Running the Lambda project

1. `npm i`
2. Create an AWS account if you don't have one.
3. Create an IAM Role: `meetnearme-bot`.
4. [Optional] Assign IAM User to User group: `meetnearme`
5. Under the IAM Role > Select the **Permissions** Tab
6. Select _Attach existing policies directly_.
7. Search for **AdministratorAccess** and select the policy by checking the checkbox, then select **Next**.
8. Click **Add Permissions**
9. Navigate back to the IAM Role
10. Go to **Security Credentials** Tab
11. Select **Create access key**
12. Select **Other** and select **Next**
13. Optionally add a tag and select **Create Access Key**
14. Create an `.env` file using `.env.example` to add needed keys.
15. Run `brew install awscli` in the terminal to install AWS CLI
16. Run `aws configure` to [Authorize SST via AWS CLI](https://sst.dev/chapters/configure-the-aws-cli.html)
17. Run `npm run dev` to run the Go Lambda Gateway V2 server locally, proxied through
   Lambda to your local

### Generate Go templates from \*.templ files

1. Add to your `.zshrc` / `.bashrc`

```
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

2. `go install github.com/a-h/templ`
3. Run `templ generate`

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

## Project Maintenance

### Updating env vars

When updating env vars, the changes need to be made in 4 places:

1. `stacks/ApiStack.ts`
2. `.github/actions/set_aws_creds_env_vars/action.yml` (`inputs` section)
3. `.github/actions/set_aws_creds_env_vars/action.yml` (`run` section where vars
   are `echo`d)
4. `.env.example` to clarify in version control what our currently-used env vars
   are
