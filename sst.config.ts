import { SSTConfig } from 'sst';

import { StorageStack } from './stacks/StorageStack';
// import { ApiStack } from './stacks/ApiStack';
import { StaticSiteStack } from './stacks/StaticSiteStack';
// import { SeshuFunction } from './stacks/SeshuFunction';

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
      timeout: '30 seconds',
    });
    app.stack(StaticSiteStack).stack(StorageStack);
    // .stack(SeshuFunction)
    // .stack((stackContext) => ApiStack({ ...stackContext, app }));
    // .stack(RdsStack)
    app.addDefaultFunctionPermissions(['dynamodb:*']);
  },
} satisfies SSTConfig;
