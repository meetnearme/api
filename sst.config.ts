import { SSTConfig } from 'sst';
import { Api, use } from 'sst/constructs';

import { StorageStack } from './stacks/StorageStack';
import { ApiStack } from './stacks/ApiStack';
import { StaticSiteStack } from './stacks/StaticSiteStack';

export default {
  config(_input) {
    return {
      name: 'meetnearme-go-fullstack',
      region: 'us-east-1',
    };
  },
  stacks(app) {
    if (process.env.GIT_BRANCH_NAME) {
      app.stage = process.env.GIT_BRANCH_NAME;
    }
    // oddly, order matters here, don't switch the order
    app.setDefaultFunctionProps({
      runtime: 'go',
    });
    app.stack(StorageStack).stack(ApiStack).stack(StaticSiteStack);
    app.addDefaultFunctionPermissions({
      permissions: ['dynamodb:*'],
    });
  },
} satisfies SSTConfig;
