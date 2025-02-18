import * as path from 'node:path';
import { promises as fs } from 'node:fs';
import { RDSDataService } from 'aws-sdk';
import { SecretsManager } from 'aws-sdk';

// Get the RDS secret ARN and database details from environment variables
const { RDS_SECRET_ARN, DATABASE_NAME, CLUSTER_ARN } = process.env;

// Fetch the secret containing database credentials from Secrets Manager
async function getDbConfig() {
  if (!RDS_SECRET_ARN) {
    throw new Error("RDS_SECRET_ARN environment variable is not set");
  }

  const secretsManager = new SecretsManager();

  // Fetch the secret value using the secret ARN
  const secretValue = await secretsManager.getSecretValue({ SecretId: RDS_SECRET_ARN }).promise();

  // Parse the secret value (usually it's stored as a JSON string)
  if (!secretValue.SecretString) {
    throw new Error("No secret string found in the RDS secret");
  }

  const secret = JSON.parse(secretValue.SecretString);
  console.log("Secret parsed from SecretValue", secret);

  return {
    host: secret.host,
    port: secret.port,
    database: DATABASE_NAME || secret.dbname,
    user: secret.username,
    password: secret.password,
  };
}

// Execute SQL statement using RDS Data API
async function executeSqlStatement(sql) {
  const rdsData = new RDSDataService();
  const params = {
    resourceArn: process.env.RDS_RESOURCE_ARN, // Your RDS cluster ARN
    secretArn: process.env.RDS_SECRET_ARN, // Your RDS secret ARN
    database: process.env.DATABASE_NAME,
    sql,
  };

  console.log("CLUSTER_ARN:", process.env.RDS_RESOURCE_ARN);
  console.log("RDS_SECRET_ARN:", process.env.RDS_SECRET_ARN);
  console.log("DATABASE_NAME:", process.env.DATABASE_NAME);


  try {
    const result = await rdsData.executeStatement(params).promise();
    console.log("SQL executed successfully:", result);
  } catch (error) {
    console.error("Failed to execute SQL statement:", error);
    throw error;
  }
}

// Run migrations from SQL files
async function runMigrations() {
  const migrationFiles = [
    path.join(__dirname, '../migrations_sql_test/001_create_event_rsvps_table.sql'),
    // path.join(__dirname, '../migrations_sql_test/002_drop_event_rsvps_table.sql'),
    path.join(__dirname, '../migrations_sql_test/003_create_purchasables_table.sql'),
    // path.join(__dirname, '../migrations_sql_test/004_drop_purchasables_table.sql'),
    path.join(__dirname, '../migrations_sql_test/005_create_users_table.sql'),
    // path.join(__dirname, '../migrations_sql_test/006_drop_users_table.sql'),
  ];

  for (const file of migrationFiles) {
    const sql = await fs.readFile(file, 'utf8');
    await executeSqlStatement(sql);
  }
}

async function listTables() {
  const sql = `
    SELECT table_name
    FROM information_schema.tables
    WHERE table_schema = 'public';
  `;

  console.log("Executing SQL to list tables:", sql);
  await executeSqlStatement(sql);
}

export async function main() {
  console.log("Starting migration...");
  try {
    await runMigrations();

    // List all tables in the database
    await listTables();

    return {
      statusCode: 200,
      body: JSON.stringify({ message: 'Migrations ran successfully' }),
    };
  } catch (error) {
    return {
      statusCode: 500,
      body: JSON.stringify({ message: 'Migrations failed', error: error.message }),
    };
  }
}

