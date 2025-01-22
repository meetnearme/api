import { StackContext, Table } from 'sst/constructs';

export function StorageStack({ stack }: StackContext) {
  // Create the `Registrations` table
  //
  const eventRsvpsTable = new Table(stack, 'EventRsvps', {
    fields: {
      id: 'string',
      userId: 'string',
      eventId: 'string',
      eventSourceId: 'string',
      eventSourceType: 'string',
      status: 'string',
      createdAt: 'number',
      updatedAt: 'number',
    },
    primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
    globalIndexes: {
      userIdGsi: { partitionKey: 'userId', sortKey: 'eventId' },
    },
  });

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

  const registrationsTable = new Table(stack, 'Registrations', {
    fields: {
      eventId: 'string',
      userId: 'string',
      responses: 'string', // this is an array, no type for arrays
      createdAt: 'number',
      updatedAt: 'number',
    },
    primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
    globalIndexes: {
      userIdGsi: { partitionKey: 'userId', sortKey: 'eventId' },
    },
  });

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

  // needs concept of allowed competitors, these must be backed by a real userId
  // and there must be a pool of them
  // access pattern needs matchup style and optional matching algorithm
  // url path needs
  const competitionConfigTable = new Table(stack, 'CompetitionConfig', {
    fields: {
      id: 'string',
      primaryOwner: 'string',
      auxilaryOwners: 'string', // JSON array
      eventIds: 'string', // jSON array
      name: 'string',
      moduleType: 'string', // KARAOKE, BOCCE
      scoringMethod: 'string', // POINTS, VOTES, etc
      rounds: 'string', // JSON string array of round configs
      competitors: 'string', // JSON string array of competitor IDs
      status: 'string', // DRAFT, ACTIVE, COMPLETE
      createdAt: 'number',
      updatedAt: 'number'
    },
    primaryIndex: { partitionKey: 'primaryOwner', sortKey: 'id' }
  });

  // This table needs the concept of sub rounds
  // this should reference another round (act in our parlance)
  // PartiQL
  // Need to have possibly sub round in PK
  const competitionRoundsTable = new Table(stack, 'CompetitionRounds', {
    fields: {
      // owner is the primary owner
      PK: 'string', // OWNER_<ownerId>
      SK: 'string', // COMPETITION_<competitionId>_ROUND_<roundNumber>
      ownerId: 'string',
      eventId: 'string',
      roundName: 'string',
      roundNumber: 'number',
      competitorA: 'string', // user
      competitorAScore: 'number',
      competitorB: 'string',
      competitorBScore: 'number',
      matchup: 'string', // <competitorA>_<competitorB> - userId
      status: 'string', // ACTIVE, COMPLETE, CANCELLED, PENDING
      competitors: 'string', // JSON string array
      parentRoundId: 'string', // for sub-rounds/acts
      isPending: 'string', // bool (use to hold multiple rounds until reveal)
      isVotingOpen: 'string', // bool
      createdAt: 'number',
      updatedAt: 'number',
    },
    primaryIndex: { partitionKey: 'PK', sortKey: 'SK' },
    globalIndexes: {
      belongsToEvent: { partitionKey: 'PK', sortKey: 'eventId' },
    },
  });


  // ephemeral
  // purchases are gate to the waiting room.
  // TODO
  const competitionWaitingRoomTable = new Table(stack, 'CompetitionWaitingRoom', {
    fields: {
      competitionId: 'string',
      userId: 'string',
      purchaseId: 'string',
      TTL: 'number', // 3 days
    },
    primaryIndex: { partitionKey: 'competitionId', sortKey: 'userId' },
    timeToLiveAttribute: 'TTL'
  });

  const votesTable = new Table(stack, 'Votes', {
    fields: {
      PK: 'string', // EVENT_<eventId>_ROUND_<roundNumber>
      SK: 'string', // USEr_<userId>  who is voting
      competitorId: 'string', // who you vote for
      competitionId: 'string',
      voteValue: 'number',
      createdAt: 'number',
      updatedAt: 'number',
      TTL: 'number'
    },
    primaryIndex: { partitionKey: 'PK', sortKey: 'SK' },
    timeToLiveAttribute: 'TTL'
  });


  return {
    registrationsTable,
    registrationFieldsTable,
    seshuSessionsTable,
    // purchasesTable, // deprecated
    purchasesTableV2,
    purchasablesTable,
    eventRsvpsTable,
    competitionConfigTable,
    competitionRoundsTable,
    votesTable,
    competitionWaitingRoomTable
  };
}
