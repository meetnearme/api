import { RDS, StackContext } from "sst/constructs";

export function RdsStack({ stack }: StackContext) {
  const DATABASE = "MeetnearmeRdsDB";

  const cluster = new RDS(stack, "Cluster", {
    engine: "postgresql13.9",
    defaultDatabaseName: DATABASE,
    migrations: "services/migrations",
  });

  stack.addOutputs({
    SecretArn: cluster.secretArn,
    ClusterIdentifier: cluster.clusterIdentifier,
    ClusterArn: cluster.clusterArn,
  });

  return {
    cluster
  };
}

// aws rds-data execute-statement \
//   --resource-arn "arn:aws:rds:us-east-1:451093494546:cluster:brandontripp-meetnearme-go-fullstack-cluster" \
//   --secret-arn  "arn:aws:secretsmanager:us-east-1:451093494546:secret:ClusterSecret26E15F5B-kwhpmQBCBE6A-0hKjEN" \
//   --database "MeetnearmeRdsDB" \
//   --sql "SHOW TABLES;" \
//   --region "us-east-1"
//
// psql \
//    --host="brandontripp-meetnearme-go-fullstack-cluster.cluster-cvkg2iaki3kv.us-east-1.rds.amazonaws.com" \
//    --port=5432 \
//    --username="postgres" \
//    --password \
//    --dbname="MeetnearmeRdsDB"

