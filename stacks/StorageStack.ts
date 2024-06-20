import { StackContext, Table } from 'sst/constructs';

export function StorageStack({ stack }: StackContext) {
  // TODO: add s3 bucket stack here
  // https://sst.dev/chapters/create-an-s3-bucket-in-sst.html

  // Create the `Events` table
  const table = new Table(stack, 'Events', {
    fields: {
      id: 'string',
      name: 'string',
      description: 'string',
      datetime: 'string',
      address: 'string',
      zipCode: 'string',
      country: 'string',
      latitude: 'number',
      longitude: 'number',
      zOrderIndex: 'binary',
    },
    primaryIndex: { partitionKey: 'zOrderIndex', sortKey: 'datetime' },
  });

  return {
    table,
  };
}
