# Meet Near Me API

## Getting Started

### Running the local SAM dynamodb docker container

1. `$ docker compose build`
1. `$ docker compose up`

### Running the Lambda project

1. `npm i`
1. Create an AWS account if you don't have one.
1. [Create an IAM User](https://sst.dev/chapters/create-an-iam-user.html)
1. Export `aws_access_key_id` and `aws_secret_access_key` env variables.
1. Run `brew install awscli` in the terminal to install AWS CLI
1. Run `aws configure` to
   [Authorize SST via AWS CLI](https://sst.dev/chapters/configure-the-aws-cli.html)
   through Lambda to your local
1. Create a `.env` file in the root directory with the necessary environment
   variables. Here's an example:

```
MEETNEARME_TEST_SECRET=anything
SCRAPINGBEE_API_KEY=ask_for_key
STATIC_BASE_URL='http://localhost:3001/static'
USE_REMOTE_DB=true
```

1. Run `npm run dev` to run the Go Lambda Gateway V2 server locally, proxied
   through Lambda to your local
1. Alternatively, you can run `npm run dev-remote-db` to run the project with a
   remote DynamoDB instance instead of the local Docker container.

### Generate Go templates from \*.templ files

1. Add to your `.zshrc` / `.bashrc`

```
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

1. `go install github.com/a-h/templ/cmd/templ@latest`
1. Run `templ generate`

### Validate home page is working + data-connected

`npm run dev` should finish with an AWS endpoint, hitting that endpoint should
show a list of events in that particular stage's dynamoDb table

### Add your AWS local deployment URL to Zitadel config

For auth to work, you must add your AWS local deployment's URL to Zitadel's
callback URLs
[our app-specific redirect settings](https://meet-near-me-production-8baqim.zitadel.cloud/ui/console/projects/273257176187855242/apps/273257486885118346)

1. Add your AWS deployment URL to `Redirect URIs` and suffix it like
   `https://{instance-id}.execute-api.us-east-1.amazonaws.com/auth/callback`
1. Add your AWS deployment URL to `Post- Logout URIs`, your deployment URL looks
   like this `https://{instance-id}.execute-api.us-east-1.amazonaws.com`

## Validating Event Basic end Points

### API Example Curl Requests

`curl <AWS URL from npm run dev>/api/event` - list table Events
`curl -X POST -H 'Content-Type: application/json' -d '{"name": "Chess Tournament", "description": "Join the junior chess tournament to test your abilities", "datetime": "2024-03-13T15:07:00", "address": "15 Chess Street", "zip_code": "84322", "country": "USA"}' <AWS URL from npm run dev>/api/event` -
insert new event

### Reference for interacting with dynamodb from aws cli v2

https://awscli.amazonaws.com/v2/documentation/api/latest/reference/dynamodb/index.html

## Project Maintenance

### Rebuilding the templ binary

If you see an error like
`(!) templ version check failed: generator v0.2.513 is older than templ version v0.2.648 found in go.mod file, consider upgrading templ CLI`,
you need to update the `templ` go binary

1. `go install github.com/a-h/templ/cmd/templ@latest`

### Updating env vars

When updating env vars, the changes need to be made in 4 places:

1. `stacks/ApiStack.ts`
1. `.github/actions/set_aws_creds_env_vars/action.yml` (`inputs` section)
1. `.github/actions/set_aws_creds_env_vars/action.yml` (`run` section where vars
   are `echo`d)
1. `.env.example` to clarify in version control what our currently-used env vars
   are

### Setting up AWS DNS in Route53 with Custom Domain names for API Gateway

For `*.meetnear.me` and `*.devnear.me` there is some custom configuration
required at the API Gateway level, DNS nameserver level, and Route53
configuration level

1. **DNS Level** - at the time of writing, the `*.me` TLD can't be registered
   through Amazon, so it's handled through Namecheap.com.
   1. First, go to Route53 in AWS console and add a new "Hosted Zone" (we'll use
      `devnear.me` as an example)
   1. In the
      [list view](https://us-east-1.console.aws.amazon.com/route53/v2/hostedzones?region=us-east-1#ListRecordSets/Z06752732TZBTZ1LBFAWP),
      look for `Type: NS` and copy the `Value`s to Namecheap.com under the
      [admin tab](https://ap.www.namecheap.com/domains/domaincontrolpanel/devnear.me/domain)
      for `devnear.me`
1. **API Gateway / Route53 Level** - To map the DNS to Route53 (covered in the
   next step), you must first configure at the API Gateway level
   1. First, go to API Gateway >
      [Custom Domain Names](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names?region=us-east-1)
      and click `Create`.
   1. Enter the new domain, in our case `devnear.me`
   1. Under `ACM Certificate` if this is a new domain, you might need to click
      [Create a new ACM Certificate](https://us-east-1.console.aws.amazon.com/acm/home?region=us-east-1)
   1. After initiating the certificate creation, you'll be taken to the AWS
      [cert admin panel](https://us-east-1.console.aws.amazon.com/acm/home?region=us-east-1#/certificates/c5840d8f-9937-4d49-abdc-83f2c5e3609c)
      where you need to click `Create Records in Route53` to verify domain
      ownership for the cert
   1. Go to the `API Gateway` >
      [Custom Domains](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names?api=unselected&region=us-east-1)
      in the AWS console
   1. Click `Create`
   1. Choose the newly created (and now verified via the Route53 records added
      above) cert
   1. Go to `devnear.me` (your newly created Custom Domain Name) >
      [Configure API Mappings](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names/api-mappings?api=unselected&domain=devnear.me&region=us-east-1)
   1. Set `API` value to the Cloudformation resource you want to map to
      `devnear.me`
   1. Go back to Route53 console to confirm the mapped `A` records are set
      correctly. If they are, the `Value` for the `A` record will be (slightly
      confusingly) `d-<alpha-numeric>.execute-api.us-east-1.amazonaws.com`. This
      should be different from the `ApiEndpoint` seen in Github Actions output
      for the deployment, which typically looks like
      `ApiEndpoint: https://<alpha-numeric>.execute-api.us-east-1.amazonaws.com`.
      The alpha-numeric characters will not match, and the correct `A` record
      should be prefixed with `d-`
