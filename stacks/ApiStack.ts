import { Api, StackContext, use } from 'sst/constructs';
import { StorageStack } from './StorageStack';

export function ApiStack({ stack }: StackContext) {
  const { table } = use(StorageStack);

  console.log(
    '~process.env.MEET_NEARME_TEST_SECRET',
    process.env.MEET_NEARME_TEST_SECRET,
  );
  console.log('~process.env.GIT_BRANCH_NAME', process.env.GIT_BRANCH_NAME);
  console.log(
    '~process.env.GIT_BRANCH_NAME ?? "nope"',
    process.env.GIT_BRANCH_NAME ?? 'nope',
  );

  const api = new Api(stack, 'api', {
    ...(process.env.GIT_BRANCH_NAME
      ? { name: `prod-${process.env.GIT_BRANCH_NAME}` }
      : {}),
    defaults: {
      function: {
        // Bind the table name to our API
        bind: [table],
        environment: {
          MEETNEARME_TEST_SECRET: process.env.MEETNEARME_TEST_SECRET,
          ZENROWS_API_KEY: process.env.ZENROWS_API_KEY,
        },
      },
    },
    routes: {
      'GET /': 'functions/lambda',
      'POST /': 'functions/lambda',
    },
  });

  console.log(`api.name: ${api.name}`);

  stack.addOutputs({
    ApiEndpoint: api.url,
  });

  return { api };
}
