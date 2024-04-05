import { Api, StackContext, use } from 'sst/constructs';
import envVars from './shared/env';
import { StaticSiteStack } from './StaticSiteStack';
import { StorageStack } from './StorageStack';
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
          ...envVars,
        },
      },
    },
    routes: {
      $default: 'functions/gateway',
    },
  });

  stack.addOutputs({
    ApiEndpoint: api.url,
  });

  return { api };
}
