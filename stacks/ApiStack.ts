import { Api, StackContext, use } from 'sst/constructs';
import { StorageStack } from './StorageStack';

export function ApiStack({ stack }: StackContext) {
  const { table } = use(StorageStack);

  stack.addDefaultFunctionPermissions({
    permissions: ['dynamodb:*'],
  });

  stack.setDefaultFunctionProps({
    runtime: 'go',
  });

  const api = new Api(stack, 'api', {
    defaults: {
      function: {
        // Bind the table name to our API
        bind: [table],
        environment: {
          MEETNEARME_TEST_SECRET: process.env.MEETNEARME_TEST_SECRET,
        },
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
