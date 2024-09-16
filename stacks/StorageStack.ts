import { StackContext, Table } from 'sst/constructs';

export function StorageStack({ stack }: StackContext) {
  // Create the `SeshuSessions` table
  const seshuSessionsTable = new Table(stack, 'SeshuSessions', {
    fields: {
      ownerId: 'string',
      url: 'string',
      urlDomain: 'string',
      urlPath: 'string',
      urlQueryParams: 'string',
      locationLatitude: 'number',
      locationLongitude: 'number',
      locationAddress: 'string',
      html: 'string',
      status: 'string',
      eventCandidates: 'string', // this is an array, but there is no type for arrays
      eventValidations: 'string', // this is an array, but there is no type for arrays
      expireAt: 'number',
      createdAt: 'number',
      updatedAt: 'number',
    },
    primaryIndex: { partitionKey: 'url' },
    timeToLiveAttribute: 'expireAt',
  });

  return {
    seshuSessionsTable,
  };
}
