import { RDS, StackContext, Function } from 'sst/constructs';
import * as customResources from 'aws-cdk-lib/custom-resources';
import * as iam from 'aws-cdk-lib/aws-iam';

export function RdsStack({ stack, app }: StackContext) {
  const DATABASE = 'MeetnearmeRdsDB';

  // Generate a unique identifier based on the stack name and stage
  const uniqueId = `${app.name}-${app.stage}-${stack.stackName}`.toLowerCase();

  // Ensure the identifier is not longer than 63 characters, starts with a letter, and doesn't end with a hyphen
  const clusterIdentifier = `rds-${uniqueId}`
    .replace(/[^a-z0-9-]/g, '-') // Replace any non-alphanumeric characters with hyphens
    .replace(/^[^a-z]/, 'rds') // Ensure it starts with a letter
    .replace(/-+/g, '-') // Replace multiple consecutive hyphens with a single hyphen
    .replace(/-$/, '') // Remove trailing hyphen if present
    .slice(0, 63);

  // Create RDS cluster
  const cluster = new RDS(stack, 'Cluster', {
    engine: 'postgresql13.9',
    defaultDatabaseName: DATABASE,
    migrations: 'services/migrations',
    cdk: {
      cluster: {
        clusterIdentifier: clusterIdentifier,
      },
    },
  });

  stack.addOutputs({
    SecretArn: cluster.secretArn,
    ClusterIdentifier: cluster.clusterIdentifier,
    ClusterArn: cluster.clusterArn,
  });

  return { cluster };
}
