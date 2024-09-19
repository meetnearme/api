import { RDS, StackContext, Function } from "sst/constructs";
import * as customResources from 'aws-cdk-lib/custom-resources';
import * as iam from 'aws-cdk-lib/aws-iam';

export function RdsStack({ stack }: StackContext) {
  const DATABASE = "MeetnearmeRdsDB";

  // Create RDS cluster
  const cluster = new RDS(stack, "Cluster", {
    engine: "postgresql13.9",
    defaultDatabaseName: DATABASE,
    migrations: "services/migrations",
  });

  // Create Lambda function for running migrations
  const migrationFunction = new Function(stack, "RunMigrationsFunction", {
    handler: 'services/functions/runMigrations.main',
    environment: {
      RDS_SECRET_ARN: cluster.secretArn,
      RDS_RESOURCE_ARN: cluster.clusterArn,
      DATABASE_NAME: DATABASE,
    },
    runtime: "nodejs18.x",
    permissions: [cluster],
    timeout: 200,
    copyFiles: [
      { from: "services/migrations_sql_test/001_create_event_rsvps_table.sql" },
      { from: "services/migrations_sql_test/003_create_purchasables_table.sql" },
      { from: "services/migrations_sql_test/005_create_users_table.sql" },

    ],
  });

  // Add permissions for Lambda to read secrets from Secrets Manager
  migrationFunction.attachPermissions([
    new iam.PolicyStatement({
      actions: ["secretsmanager:GetSecretValue"],
      resources: [cluster.secretArn, cluster.clusterArn], // Allow access to the secret
    }),
  ]);

  // Define IAM role for the custom resource
  const migrationTriggerRole = new iam.Role(stack, "MigrationTriggerRole", {
    assumedBy: new iam.ServicePrincipal("lambda.amazonaws.com"),
    managedPolicies: [
      iam.ManagedPolicy.fromAwsManagedPolicyName(
        "service-role/AWSLambdaBasicExecutionRole"
      ),
    ],
  });

  // Add permissions for the custom resource to invoke the Lambda function
  migrationTriggerRole.addToPolicy(new iam.PolicyStatement({
    actions: ['lambda:InvokeFunction'],
    resources: [migrationFunction.functionArn],
  }));

  // Custom resource to trigger Lambda
  const migrationTrigger = new customResources.AwsCustomResource(stack, 'MigrationTrigger', {
    onCreate: {
      service: 'Lambda',
      action: 'invoke',
      parameters: {
        FunctionName: migrationFunction.functionName,
        InvocationType: 'RequestResponse',
      },
      physicalResourceId: customResources.PhysicalResourceId.of(Date.now().toString()), // Unique ID
    },
    policy: customResources.AwsCustomResourcePolicy.fromStatements([
      new iam.PolicyStatement({
        actions: ['lambda:InvokeFunction'],
        resources: [migrationFunction.functionArn], // Permission to invoke migration Lambda
      }),
    ]),
    role: migrationTriggerRole, // Attach the role to the custom resource
  });

  migrationTrigger.node.addDependency(cluster); // Ensure RDS is set up before migration trigger

  stack.addOutputs({
    SecretArn: cluster.secretArn,
    ClusterIdentifier: cluster.clusterIdentifier,
    ClusterArn: cluster.clusterArn,
  });

  return { cluster };
}

