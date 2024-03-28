import { Api, StackContext, use } from 'sst/constructs';
import { StorageStack } from './StorageStack';
import { StaticSiteStack } from './StaticSiteStack';
import { HostedZone } from 'aws-cdk-lib/aws-route53';
import { Certificate } from 'aws-cdk-lib/aws-certificatemanager';

export function ApiStack({ stack }: StackContext) {
  const { table } = use(StorageStack);
  const { staticSite } = use(StaticSiteStack);

  const api = new Api(stack, 'api', {
    defaults: {
      function: {
        // Bind the table name to our API
        bind: [table],
        environment: {
          MEETNEARME_TEST_SECRET: process.env.MEETNEARME_TEST_SECRET,
          ZENROWS_API_KEY: process.env.ZENROWS_API_KEY,
          // STATIC_BASE_URL is a special case because the value comes from
          // `sst deploy` at runtime and then gets set as an environment variable
          STATIC_BASE_URL: process.env.STATIC_BASE_URL ?? staticSite.url,
          // ROUTE53_HOSTED_ZONE_ID omitted because it's only used in deployment
          // ROUTE53_HOSTED_ZONE_NAME omitted because it's only used in deployment
        },
      },
    },
    customDomain: {
      isExternalDomain: true,
      domainName: 'meetnear.me',
      cdk: {
        certificate: Certificate.fromCertificateArn(
          stack,
          'MeetNearMeCert',
          process.env.AWS_MEETNEARME_CERT_ARN,
        ),
        hostedZone: HostedZone.fromHostedZoneAttributes(
          stack,
          'MeetNearMeZone',
          {
            hostedZoneId: process.env.ROUTE53_HOSTED_ZONE_ID,
            zoneName: process.env.ROUTE53_HOSTED_ZONE_NAME,
          },
        ),
      },
    },
    routes: {
      'GET /': 'functions/lambda',
      'POST /': 'functions/lambda',
    },
  });

  stack.addOutputs({
    ApiEndpoint: api.url,
  });

  return { api };
}
