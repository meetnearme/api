import { StackContext, Table } from 'sst/constructs';
import { DynamoDBClient, DescribeTableCommand } from '@aws-sdk/client-dynamodb';

export async function StorageStack({ stack }: StackContext) {
  const client = new DynamoDBClient({});

  async function tableExists(tableName: string) {
    try {
      await client.send(new DescribeTableCommand({ TableName: tableName }));
      return true;
    } catch (error) {
      if (error.name === 'ResourceNotFoundException') {
        return false;
      }
      throw error;
    }
  }

  const registrationsTableName = 'Registrations';
  const registrationsTableExists = await tableExists(registrationsTableName);

  // Create the `Registrations` table
  let registrationsTable;
  if (!registrationsTableExists) {
    registrationsTable = new Table(stack, tableName, {
      fields: {
        eventId: 'string',
        userId: 'string',
        responses: 'string', // this is an array, no type for arrays
        createdAt: 'number',
        updatedAt: 'number',
        updatedBy: 'string',
      },
      primaryIndex: { partitionKey: 'eventId', sortKey: 'userId' },
    });
  }

  const registrationFieldsTableName = 'RegistrationFields';
  const registrationFieldsTableExists = await tableExists(
    registrationFieldsTableName,
  );

  // Create the `Registrations` table
  let registrationFieldsTable;
  if (!registrationFieldsTableExists) {
    const registrationFieldsTable = new Table(
      stack,
      registrationFieldsTableName,
      {
        fields: {
          eventId: 'string',
          fields: 'string', // this is an array of registrationFields
          createdAt: 'number',
          updatedAt: 'number',
          updatedBy: 'string',
        },
        primaryIndex: { partitionKey: 'eventId' },
      },
    );
  }

  const SeshuSessionsTableName = 'SeshuSessions';
  const SeshuSessionsTableExists = await tableExists(SeshuSessionsTableName);

  // Create the `Registrations` table
  let seshuSessionsTable;
  if (!SeshuSessionsTableExists) {
    const seshuSessionsTable = new Table(stack, SeshuSessionsTableName, {
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
  }

  return {
    registrationsTable,
    registrationFieldsTable,
    seshuSessionsTable,
  };
}
