# Meet Near Me API

### Running the Lambda project

1. `npm i`
1. Create an AWS account if you don't have one.
1. [Create an IAM User](https://sst.dev/chapters/create-an-iam-user.html)
1. Export `aws_access_key_id` and `aws_secret_access_key` env variables.
1. Run `brew install awscli` in the terminal to install AWS CLI
1. Run `brew install go` 
1. Run `aws configure` to
   [Authorize SST via AWS CLI](https://sst.dev/chapters/configure-the-aws-cli.html)
   through Lambda to your local environment
1. Create a `.env` file in the root directory with the necessary environment
   variables found in [.env.example](.env.example).
1. Copy the file `functions/gateway/helpers/cloudflare_locations_template` to  => `functions/gateway/helpers/cloudflare_locations.go` creating a new file with the `.go` extension
1. Find the string 
    ```
    const cfLocations = `<replace me>`
    ```
    in the `cloudflare_locations.go` file and replace the `<replace me>` (keep the wrapping backtick characters) with the JSON from [speed.cloudflare.com/locations](https://speed.cloudflare.com/locations)... this file is used to intercept incoming request headers and look at the `Request['Headers']['Cf-Ray']` string, which terminates with a 3 letter code like `EWR` which correlates with an `iata` airport code. This is the Cloudflare datacenter serving the request
1. Run `npm run dev:sst` to run the initial AWS project deployment. **NOTE** that you will NOT use the same command after the initial deployment of your project
1. Give the project a an SST `stage` name, it can be anything 
1. Once you see `✔  Deployed:` ... look for `ApexUrl: https://<abcxyz>.execute-api.us-east-1.amazonaws.com` where `abcxyz` is a hash unique to your project
1. Copy this URL and open it in a browser to test that the project runs
1. Use CTRL+C to stop the project 
1. **Restart the project with `npm run dev`** – you will use this command for all **subsequent** attempts to start the project) in order to have the project run the following:
   1. **SST Go Lambda server with hot reload / Go rebuild** – this proxies real AWS resources pointing back to your local Go server through Lambda, and simplifies / sidesteps the complexity of needing to use proxies to ensure harmonious server behaviors liike same-origin and other typical local dev hurdles. ⚠️ Unlike React / JS frontends, since our Go monolith requires a rebuild whenever a template or Go file changes, check your terminal to get a feel for typical "rebuild time". A common issue is to have the server die often when you make a request while it's rebuilding. To avoid this, you really only need to wait maybe about 2 seconds for the rebuild to happen. Once it's finished, you'll be fine, but if you refresh a page `0.25s` after a change initiates a rebuild you might see something like `Error: spawn /Users/bfeister/dev/meetnearme-api/.sst/artifacts/c8ae08557883d489e35129f0feb436ead1e1695501/bootstrap ENOENT` and then `[serve-static] http-server stopped.`
   1. **Local static asset server**
   1. **Local node.js based `templ`** – our [golang templating engine pkg](https://templ.guide/)) watch script to rebuild templates as they're modified
   1. **Local tailwind watcher**  – this includes some complexity, because we use a hashing algorithim that updates `layout.templ` with a new hash value whenever the tailwind styles in the template change for cache-busting when new deployments are pushed to prod. The dev experience here is a work-in-progress. I've noticed that sometimes updates to `layout.templ` can require a FULL stop / restart of the `npm run dev` processes to take effect.
1. Add your `APEX_URL` from your local deployment to the allow lists for both **Redirect** and **Post Logout URls** in Zitadel under **Redirect Settings** in [the admin UI](https://meet-near-me-production-8baqim.zitadel.cloud/ui/console/projects/273257176187855242/apps/273257486885118346) following the existing URL path schema for both. If you don't have admin access in Zitadel, ask someone on the team   

### Generate Go templates from \*.templ files

1. Add to your `.zshrc` / `.bashrc`

```
export GOPATH=$HOME/go
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

## Validating Event Basic Eendpoints

### API Example Curl Requests

- Add a new event - `curl <AWS URL from npm run dev>/api/event` - list table
  Events
  `curl -X POST -H 'Content-Type: application/json' --data-raw $'{"events":[{"eventOwners":["123"],"name":"Espanola Lowriders Good Friday Rally & Bar Crawl","description":"Join us in the low rider capital of the world while we hit up all TWO of our local bars\u0021 You haven\'t seen a real lowrider if you haven\'t visited Espanola\u0021","startTime":"2025-02-15T18:30:00Z","address":"Espanola, NM","lat": 36.015303,"long":-106.066063}]}' <AWS URL from npm run dev>/api/event`

### Reference for interacting with dynamodb from aws cli v2

https://awscli.amazonaws.com/v2/documentation/api/latest/reference/dynamodb/index.html

## Project Maintenance

### Rebuilding the templ binary

If you see an error like
`(!) templ version check failed: generator v0.2.513 is older than templ version v0.2.648 found in go.mod file, consider upgrading templ CLI`,
you need to update the `templ` go binary

1. `go install github.com/a-h/templ/cmd/templ@latest`

### Updating env vars

For an overview of our current env vars with an explanation of each, look at
[.env.example](.env.example)

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

## Legacy Details

### Running the local SAM dynamodb docker container

1. `$ docker compose build`
1. `$ docker compose up`
