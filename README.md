# Meet Near Me API

## Running the Docker Project

1. Install Docker via [`Download Docker Desktop button here`](https://www.docker.com/products/docker-desktop/)
2. Get the latest `.env` file variables from someone on the team
3. Install npm
4. Git clone this repo
5. `npm install`
6. Open two terminal tabs
7. In the first, execute `npm run dev:docker:rebuild`, this will build the multi-container project (you can view the various containers [here](https://github.com/meetnearme/api/blob/develop/docker-compose.yml)) and expose the main project on `localhost:8000`
8. In the second, execute `npm run dev:watchers`, this will run a few things
    1. Tailwind (with a special watcher that can update the css hash in `layout.templ`
    2. The `*.templ` template file watcher and a trigger to compile new Go builds
    3. local npm static asset server

## (⚠️ Deprecaated) Running the Lambda Project

### Prerequisites

1. **Install dependencies:**
   ```bash
   npm install
   ```
2. **[Create an AWS account](https://signin.aws.amazon.com/signup?request_type=register)**, if you don’t have one.
3. **[Create an IAM User](https://sst.dev/chapters/create-an-iam-user.html)**.
4. **Export `aws_access_key_id` and `aws_secret_access_key` env variables: [How?](https://docs.aws.amazon.com/singlesignon/latest/userguide/generate-token.html)**
5. **Install AWS CLI:**
[AWS CLI Installation Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
   - macOS:
     ```bash
     brew install awscli
     ```
   - Linux (Ubuntu):
     ```bash
     sudo apt install awscli
     ```
   - Window: check the link for AWS CLI Guide

    Check version with `aws --version`
6. **Configure AWS CLI:**
   [Authorize SST via AWS CLI](https://sst.dev/chapters/configure-the-aws-cli.html)
   ```bash
   aws configure
   ```
7. **Install Go:**
[Go Installation Guide](https://go.dev/dl/)
    - macOS:
        ```bash
        brew install go
         ```
   - Linux (Ubuntu):
      *Replace <go-version> with latest stable go version below*
     ```bash
     tar -C /usr/local -xzf go<go-version>.linux-amd64.tar.gz
     export PATH=$PATH:/usr/local/go/bin
     ```
   - Window: check the link above for installation of latest stable go version

    Check version with ```go version ```

8. **Install templ:**
[templ Installation Guide](https://templ.guide/quick-start/installation)
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```
    *Note: ensure minimium templ version >= v0.2.793*

9. **Set up environment variables:**
 Create a `.env` file in the root directory based on `.env.example`.

### Initial Setup

1. **Copy template file:**
   - Copy `functions/gateway/helpers/cloudflare_locations_template` to `functions/gateway/helpers/cloudflare_locations.go`.
2. **Update Cloudflare locations:**
   - Replace `<replace me>` in the `cloudflare_locations.go` file with JSON from [speed.cloudflare.com/locations](https://speed.cloudflare.com/locations):
     ```go
     const cfLocations = `<JSON content>`
     ```
   *Note*: This file is used to intercept incoming request headers and look at the `Request['Headers']['Cf-Ray']` string, which terminates with a 3 letter code like `EWR` which correlates with an `iata` airport code. This is the Cloudflare datacenter serving the request
3. **Deploy the project:**
   ```bash
   npm run sst-dev

   ```
   *NOTE: This should be run a single time after the aws credentials are setup to deploy your AWS Dynamodb tables
   *NOTE: If you are encountering problem as a Window user, due to `'NODE_ENV' is not recognized as an internal or external command` do check Project Maintenance for troubleshooting*

    **⚠️IMPORTANT:** that you will NOT use the same command after the initial deployment of your project
4. **Assign SST stage name:** Choose any name for your SST `stage`.
5. **Test the deployment:**
- Once you see `✔  Deployed:` ... look for `ApexUrl: https://<abcxyz>.execute-api.us-east-1.amazonaws.com` where `abcxyz` is a hash unique to your project
- Copy this URL and open it in a browser to test that the project runs

6. **Stop and restart the project:**
   - Stop the project using `CTRL+C`.
   - Restart with:
     ```bash
     npm run dev
     ```
    You will use ```npm run dev``` command for all subsequent attempts to start the project

7. Please look at the .env.example file for the bottom env variables. Please fill these out with the tables names found in your AWS account under dynamodb tables section.

8. Please see below for the up to date Docker commands to build the project.

- The following processes will run with `npm run dev`:
  1. **SST Go Lambda server:** Hot reload with Go rebuild.
      - This proxies real AWS resources pointing back to your local Go server through Lambda, and simplifies / sidesteps the complexity of needing to use proxies to ensure harmonious server behaviors liike same-origin and other typical local dev hurdles.
      - ⚠️ Unlike React / JS frontends, since our Go monolith requires a rebuild whenever a template or Go file changes, check your terminal to get a feel for typical "rebuild time". A common issue is to have the server die often when you make a request while it's rebuilding. To avoid this, you really only need to wait maybe about 2 seconds for the rebuild to happen. Once it's finished, you'll be fine, but if you refresh a page `0.25s` after a change initiates a rebuild you might see something like `Error: spawn /Users/bfeister/dev/meetnearme-api/.sst/artifacts/c8ae08557883d489e35129f0feb436ead1e1695501/bootstrap ENOENT` and then `[serve-static] http-server stopped.`
  2. **Static asset server:** Serves local static files.
  3. **Local node.js based `templ`:**  our [golang templating engine pkg](https://templ.guide/) watch script to rebuild templates as they're modified
  4. **Local tailwind watcher:**  this includes some complexity, because we use a hashing algorithim that updates `layout.templ` with a new hash value whenever the tailwind styles in the template change for cache-busting when new deployments are pushed to prod. The dev experience here is a work-in-progress. I've noticed that sometimes updates to `layout.templ` can require a FULL stop / restart of the `npm run dev` processes to take effect.

### Zitadel Configuration
- Add your `APEX_URL` from your local deployment to the allow lists for both **Redirect** and **Post Logout URls** in Zitadel under **Redirect Settings** in [the admin UI](https://meet-near-me-production-8baqim.zitadel.cloud/ui/console/projects/273257176187855242/apps/273257486885118346) following the existing URL path schema for both. If you don't have admin access in Zitadel, ask someone on the team

#### Add your AWS local deployment URL to Zitadel Configuration

- For auth to work, you must add your AWS local deployment's URL to Zitadel's
callback URLs [our app-specific redirect settings](https://meet-near-me-production-8baqim.zitadel.cloud/ui/console/projects/273257176187855242/apps/273257486885118346)
1. Add your AWS deployment URL to `Redirect URIs` and suffix it like
   `https://{instance-id}.execute-api.us-east-1.amazonaws.com/auth/callback`
2. Add your AWS deployment URL to `Post- Logout URIs`, your deployment URL looks
   like this `https://{instance-id}.execute-api.us-east-1.amazonaws.com`

## Generating Go Templates from `*.templ` Files

1. Add to `.zshrc` or `.bashrc`:
   ```bash
   export GOPATH=$HOME/go
   ```
2. Install templ:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```
3. Generate templates:
   ```bash
   templ generate
   ```
### Rebuilding the `templ` Binary

If you encounter an error like:

```bash
(!) templ version check failed: generator v0.2.513 is older than templ version v0.2.648 found in go.mod file, consider upgrading templ CLI
```

Update or Downgrade templ to the respective version specified:
   ```bash
   go install github.com/a-h/templ/cmd/templ@<version>
   ```
*Note: Current build is templ@v0.2.793, but might change in the future*
## Validating Home Page and Data Connectivity

- `npm run dev` should finish with an AWS endpoint, hitting that endpoint should
show a list of events in that particular stage's dynamoDb table

## Validating Event Basic Endpoints
**API Example Curl Requests:**
- Add a new event, replace `<AWS URL>`:

  ```bash
  curl -X POST -H 'Content-Type: application/json' \
  --data-raw '{"events":[{"eventOwners":["123"],"name":"Espanola Lowriders Good Friday Rally & Bar Crawl","description":"Join us in the low rider capital of the world while we hit up all TWO of our local bars! You haven\'t seen a real lowrider if you haven\'t visited Espanola!","startTime":"2025-02-15T18:30:00Z","address":"Espanola, NM","lat":36.015303,"long":-106.066063}]}' \
  <AWS URL>/api/event
  ```
### Reference for interacting with dynamodb from aws cli v2

https://awscli.amazonaws.com/v2/documentation/api/latest/reference/dynamodb/index.html
## Project Maintenance

### Window's Troubleshoot:
If you are using windows operating system, you might encouter this error during the build stage:
```
[sst] Building static site static
[sst] 'NODE_ENV' is not recognized as an internal or external command,
[sst] operable program or batch file.
[sst] ✖  Checking for changes
```
**Solution:**
1. In stacks> StaticSiteStack.ts:
    `
    buildCommand: 'NODE_ENV=production npm run tailwind:prod'
    `
    replace with
    `
    buildCommand: 'set NODE_ENV=production && npm run tailwind:prod'
    `

2. In package.json
    `
    "tailwind:prod": "NODE_ENV=production tailwindcss --postcss -i ./static/assets/global.css -o ./static/assets/styles.css --minify"
    `
    replace with
    `
    "tailwind:prod": "set NODE_ENV=production && tailwindcss --postcss -i ./static/assets/global.css -o ./static/assets/styles.css --minify"
    `
### Updating Environment Variables
1. Update env vars in the following locations:
   - `stacks/ApiStack.ts`
   - `.github/actions/set_aws_creds_env_vars/action.yml` (both `inputs` and `run` sections).
   - `.env.example` to reflect current env vars.

### Setting up AWS DNS in Route53 with Custom Domain names for API Gateway

For `*.meetnear.me` and `*.devnear.me` there is some custom configuration required at the API Gateway level, DNS nameserver level, and Route53 configuration level

**DNS Level** - at the time of writing, the `*.me` TLD can't be registered through Amazon, so it's handled through Namecheap.com,
1.  First, go to Route53 in AWS console and add a new "Hosted Zone" (we'll use`devnear.me` as an example)
2.  In the [list view](https://us-east-1.console.aws.amazon.com/route53/v2/hostedzones?region=us-east-1#ListRecordSets/Z06752732TZBTZ1LBFAWP), look for `Type: NS` and copy the `Value`s to Namecheap.com under the [admin tab](https://ap.www.namecheap.com/domains/domaincontrolpanel/devnear.me/domain) for `devnear.me`

**API Gateway / Route53 Level** - To map the DNS to Route53 (covered in the next step), you must first configure at the API Gateway level
1. First, go to API Gateway > [Custom Domain Names](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names?region=us-east-1)  and click `Create`.
2. Enter the new domain, in our case `devnear.me`
3. Under `ACM Certificate` if this is a new domain, you might need to click
    [Create a new ACM Certificate](https://us-east-1.console.aws.amazon.com/acm/home?region=us-east-1)
4.  After initiating the certificate creation, you'll be taken to the AWS
      [cert admin panel](https://us-east-1.console.aws.amazon.com/acm/home?region=us-east-1#/certificates/c5840d8f-9937-4d49-abdc-83f2c5e3609c)
      where you need to click `Create Records in Route53` to verify domain
      ownership for the cert
5. Go to the `API Gateway` > [Custom Domains](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names?api=unselected&region=us-east-1) in the AWS console
6. Click `Create`
7. Choose the newly created (and now verified via the Route53 records added
      above) cert
8. Go to `devnear.me` (your newly created Custom Domain Name) >
      [Configure API Mappings](https://us-east-1.console.aws.amazon.com/apigateway/main/publish/domain-names/api-mappings?api=unselected&domain=devnear.me&region=us-east-1)
9. Set `API` value to the Cloudformation resource you want to map to
      `devnear.me`
10. Go back to Route53 console to confirm the mapped `A` records are set
      correctly. If they are, the `Value` for the `A` record will be (slightly
      confusingly) `d-<alpha-numeric>.execute-api.us-east-1.amazonaws.com`. This
      should be different from the `ApiEndpoint` seen in Github Actions output
      for the deployment, which typically looks like
      `ApiEndpoint: https://<alpha-numeric>.execute-api.us-east-1.amazonaws.com`.
      The alpha-numeric characters will not match, and the correct `A` record should be prefixed with `d-`

## Docker Compose For Local Development. 

**note that we build the go binary locally so go must be installed, this allows for our watchGolang.js script to rebuild on changes**

Summary of this workflow:

1. Initial/Full Start (or after changes requiring rebuild):

    - npm run docker:up (builds Go app, creates/starts fresh containers, shows logs)
    - OR npm run docker:rebuild (cleans everything including data, builds, starts fresh, shows logs)

2. Temporary Pause:

    - npm run docker:stop (or docker compose stop)

3. Quick Resume (from paused state):

    - npm run docker:start (or docker compose start)

4. After docker is up and running run the command below to setup weaviate schema and then seed with the test json data:
 
    - `npm run docker:weaviate:create-schema && npm run docker:weaviate:seed-json --file=test_events_for_seeding.json`


## Legacy Details

### Running Local SAM DynamoDB Docker Container

1. Build and start:
   ```bash
   docker compose build
   docker compose up
   ```
