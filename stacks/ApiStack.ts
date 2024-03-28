import { Api, StackContext, use } from 'sst/constructs';

import { StaticSiteStack } from './StaticSiteStack';
import { StorageStack } from './StorageStack';

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
        },
      },
    },
    routes: {
      $default: 'functions/lambda'
    },
  });

  stack.addOutputs({
    ApiEndpoint: api.url,
  });

  return { api };
}
