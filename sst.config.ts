import { SSTConfig } from 'sst';
import { Api, Table } from 'sst/constructs';

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
    app.stack(function Stack({ stack }) {
      // Create the `Events` table
      const table = new Table(stack, 'Events', {
        fields: {
          Id: 'string',
          Name: 'string',
          Description: 'string',
          Datetime: 'string',
          Address: 'string',
          ZipCode: 'string',
          Country: 'string',
        },
        primaryIndex: { partitionKey: 'Id' },
      });

      const api = new Api(stack, 'api', {
        defaults: {
          function: {
            // Bind the table name to our API
            bind: [table],
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
    });
  },
} satisfies SSTConfig;
