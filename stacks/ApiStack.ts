import { Api, StackContext, use } from 'sst/constructs';
import envVars from './shared/env';
import { StaticSiteStack } from './StaticSiteStack';
import { StorageStack } from './StorageStack';
import { HostedZone } from 'aws-cdk-lib/aws-route53';
import { Certificate } from 'aws-cdk-lib/aws-certificatemanager';
import { SeshuFunction } from './SeshuFunction';

export function ApiStack({ stack }: StackContext) {
  const { eventsTable } = use(StorageStack);
  const { seshuSessionsTable } = use(StorageStack);
  const { staticSite } = use(StaticSiteStack);
  const { seshuFn } = use(SeshuFunction);

  const api = new Api(stack, 'api', {
    defaults: {
      function: {
        // Bind the eventsTable name to our API
        bind: [eventsTable, seshuSessionsTable],
        environment: {
          ...envVars,
          // ----- BEGIN -----
          // the vars below are a special case because the value comes from
          // `sst deploy` at runtime and then gets set as an environment variable
          STATIC_BASE_URL: process.env.STATIC_BASE_URL ?? staticSite.url,
          SESHU_FN_URL: process.env.SESHU_FN_URL ?? seshuFn.url,
          // ----- END -----
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
