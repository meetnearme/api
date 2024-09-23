import { Api, StackContext, use } from 'sst/constructs';
import envVars from './shared/env';
import { StaticSiteStack } from './StaticSiteStack';
import { StorageStack } from './StorageStack';
import { HostedZone } from 'aws-cdk-lib/aws-route53';
import { Certificate } from 'aws-cdk-lib/aws-certificatemanager';
import { SeshuFunction } from './SeshuFunction';
import { RdsStack } from './RdsStack';

export function ApiStack({ stack, app }: StackContext & { app: any }) {
  const { seshuSessionsTable, registrationsTable, registrationFieldsTable } = use(StorageStack);
  const { staticSite } = use(StaticSiteStack);
  const { seshuFn } = use(SeshuFunction);
  const { cluster } = use(RdsStack);

  const api = new Api(stack, 'api', {
    defaults: {
      function: {
        // Bind the eventsTable name to our API
        bind: [ seshuSessionsTable, registrationsTable, registrationFieldsTable, cluster],
        environment: {
          ...envVars,
          // ----- BEGIN -----
          // the vars below are a special case because the value comes from
          // `sst deploy` at runtime and then gets set as an environment variable
          STATIC_BASE_URL: process.env.STATIC_BASE_URL ?? staticSite.url,
          SESHU_FN_URL: process.env.SESHU_FN_URL ?? seshuFn.url,
          SST_STAGE: app.stage,
          DATABASE_NAME: "MeetnearmeRdsDB",
          RDS_CLUSTER_ARN: cluster.clusterArn,
          RDS_SECRET_ARN: cluster.secretArn,
          // ----- END -----
        },
      },
    },
  });

  let apexUrl;
  if (app.stage === 'prod') {
    apexUrl = process.env.APEX_URL;
  } else if (app.stage === 'dev') {
    apexUrl = process.env.APEX_DEV_URL;
  } else {
    apexUrl = api.url;
  }

  // $default route is added separately because we want to get `api.url` which is yielded above among
  // others, this is used by zitadel auth to redirect back to a frontend URL that matches the apex

  api.addRoutes(stack, {
    $default: {
      function: {
        handler: 'functions/gateway',
        environment: {
          APEX_URL: apexUrl,
        },
      },
    },
  });

  stack.addOutputs({
    ApexUrl: apexUrl,
    CloudformationApiUrl: api.url,
  });

  return { api };
}
