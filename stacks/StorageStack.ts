import { StackContext, Table } from 'sst/constructs';

export function StorageStack({ stack }: StackContext) {
  // Create the `Events` table
  const eventsTable = new Table(stack, 'Events', {
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

  const seshuSessionsTable = new Table(stack, 'SeshuSessions', {
    fields: {
      ownerId: 'string', // TODO: on seshu function start, check for existing seshu job
      // key via query to "jobs" table, if it exists, then return a
      // 409 error to the client, explaining it can't be added
      url: 'string',
      urlDomain: 'string',
      urlPath: 'string',
      urlQueryParams: 'string', // TODO: these need to be sorted at the API level
      // upon DB insert, to prevent duplicates
      locationLatitude: 'number', // TODO: if the "are these in the same location" checkbox in UI is not checked,
      // then `latitude` `longitude` and `address` should be null
      locationLongitude: 'number',
      locationAddress: 'string',
      html: 'string',
      status: 'string',
      eventCandidates: 'string', // this is an array, but there is no type for arrays
      eventValidations: 'string', // this is an array, but there is no type for arrays
      expireAt: 'number',
      createdAt: 'number',
      updatedAt: 'number', // TODO: consider updating `expireAt` when session is updated
    },
    primaryIndex: { partitionKey: 'url' },
    timeToLiveAttribute: 'expireAt', // TODO: must be stored in epoch format
  });

  return {
    eventsTable,
    seshuSessionsTable,
  };
}
