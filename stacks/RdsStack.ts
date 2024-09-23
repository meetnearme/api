import { RDS, StackContext, Function } from 'sst/constructs';
import * as customResources from 'aws-cdk-lib/custom-resources';
import * as iam from 'aws-cdk-lib/aws-iam';

export function RdsStack({ stack, app }: StackContext) {
  const DATABASE = 'MeetnearmeRdsDB';

  const clusterIdentifier =
    app.stage == 'prod'
      ? `prod-meetnearme-go-fullstack-cluster`
      : `dev-meetnearme-go-fullstack-cluster`;

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
