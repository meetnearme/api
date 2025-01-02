import { StackContext, Table } from 'sst/constructs';

export function StorageStack({ stack }: StackContext) {
  // 🚨 WARNING 🚨 Deprecated, do not use
  // const eventRsvpsTable = new Table(stack, 'EventRsvps', {
  //   fields: {
  //     id: 'string',
  //     userId: 'string',
  //     eventId: 'string',
  //     eventSourceId: 'string',
  //     eventSourceType: 'string',
  //     status: 'string',
  //     createdAt: 'number',
  //     updatedAt: 'number',
  //   },
  //   primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
  //   globalIndexes: {
  //     userIdGsi: { partitionKey: 'userId', sortKey: 'eventId' },
  //   },
  // });

  // const purchasesTable = new Table(stack, 'Purchases', {
  //   fields: {
  //     id: 'string',
  //     userId: 'string',
  //     eventId: 'string',
  //     status: 'string',
  //     purchasedItems: 'string',
  //     total: 'number',
  //     currency: 'string',
  //     stripeSessionId: 'string',
  //     createdAt: 'number',
  //     updatedAt: 'number',
  //   },
  //   primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
  //   globalIndexes: {
  //     userIdGsi: { partitionKey: 'userId', sortKey: 'eventId' },
  //   },
  // });

  const purchasesTableV2 = new Table(stack, 'PurchasesV2', {
    fields: {
      compositeKey: 'string',
      userId: 'string',
      eventId: 'string',
      createdAt: 'number',
      createdAtString: 'string',
      updatedAt: 'number',
      status: 'string',
      purchasedItems: 'string',
      total: 'number',
      currency: 'string',
      stripeSessionId: 'string',
      stripeTransactionId: 'string',
    },
    primaryIndex: { partitionKey: 'compositeKey' },
    globalIndexes: {
      userIdIndex: { partitionKey: 'userId', sortKey: 'createdAtString' },
      eventIdIndex: { partitionKey: 'eventId', sortKey: 'createdAtString' },
    },
  });

  const purchasablesTable = new Table(stack, 'Purchasables', {
    fields: {
      eventId: 'string',
      registrationFieldNames: 'string',
      purchasableItems: 'string',
      createdAt: 'number',
      updatedAt: 'number',
    },
    primaryIndex: { partitionKey: 'eventId' },
  });

  // 🚨 WARNING 🚨 Deprecated, do not use
  // const registrationsTable = new Table(stack, 'Registrations', {
  //   fields: {
  //     eventId: 'string',
  //     userId: 'string',
  //     responses: 'string', // this is an array, no type for arrays
  //     createdAt: 'number',
  //     updatedAt: 'number',
  //   },
  //   primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
  //   globalIndexes: {
  //     userIdGsi: { partitionKey: 'userId', sortKey: 'eventId' },
  //   },
  // });

  const registrationFieldsTable = new Table(stack, 'RegistrationFields', {
    fields: {
      eventId: 'string',
      fields: 'string', // this is an array of registrationFields
      createdAt: 'number',
      updatedAt: 'number',
      updatedBy: 'string',
    },
    primaryIndex: { partitionKey: 'eventId' },
  });

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
    // registrationsTable,  // deprecated
    registrationFieldsTable,
    seshuSessionsTable,
    // purchasesTable, // deprecated
    purchasesTableV2,
    purchasablesTable,
    // eventRsvpsTable, // deprecated
  };
}
