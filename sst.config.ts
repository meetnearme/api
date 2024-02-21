import { SSTConfig } from 'sst';
import { Api, use } from 'sst/constructs';

import { StorageStack } from './stacks/StorageStack';
import { ApiStack } from './stacks/ApiStack';

export default {
  config(_input) {
    return {
      name: 'meetnearme-go-fullstack',
      region: 'us-east-1',
    };
  },
  stacks(app) {
    app.stack(StorageStack).stack(ApiStack);
  },
} satisfies SSTConfig;
