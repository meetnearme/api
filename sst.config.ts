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
    // console.log('~TEST!!!', process.env);
    app.setDefaultFunctionProps({
      runtime: 'go',
    });
    // app.addDefaultFunctionPermissions({
    //   // permissions: ['dynamodb:PutItem', 'dynamodb:GetItem', 'dynamodb:Scan'],
    //   // attachPermissionsToPolicy: true,
    //   // attachPermissionsToRole: ['dynamodb'],
    // });
    app.stack(StorageStack).stack(ApiStack);

    //   // tableName: 'Events',
    //   // actions: ['dynamodb:PutItem', 'dynamodb:GetItem', 'dynamodb:Scan'],

    //   // maybe needed?
    //   // attachPermissionsToRole

    //   // maybe needed?
    //   // attachPermissionsToPolicy
    // });
  },
} satisfies SSTConfig;
