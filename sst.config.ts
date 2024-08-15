import { SSTConfig } from 'sst';

import { StorageStack } from './stacks/StorageStack';
import { ApiStack } from './stacks/ApiStack';
import { StaticSiteStack } from './stacks/StaticSiteStack';
import { SeshuFunction } from './stacks/SeshuFunction';
import { MarqoStack } from './stacks/MarqoStack';

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
    app
      .stack(StaticSiteStack)
      .stack(StorageStack)
      .stack(SeshuFunction)
      .stack(MarqoStack)
      .stack((stackContext) => ApiStack({ ...stackContext, app }));
    app.addDefaultFunctionPermissions(['dynamodb:*']);
  },
} satisfies SSTConfig;
