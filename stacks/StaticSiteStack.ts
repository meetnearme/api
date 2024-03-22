import { Api, StackContext, StaticSite, use } from 'sst/constructs';

import { ApiStack } from './ApiStack';

// TODO: pass `api` created from that stack in to get `api.url` here for `alternateNames`
// the problem is that `ApiStack.ts` needs `StatiSiteStack.ts` and vice versa so
// there's a need to handle dependency injection in some higher level place like
// `sst.config.ts`
export function StaticSiteStack({ stack }: StackContext) {
  // const { api } = use(ApiStack);
  const staticSite = new StaticSite(stack, 'frontend', {
    path: 'static',
    dev: {
      deploy: true,
      url: 'http://localhost:3001',
    },
    buildCommand: 'npm run tailwind:prod',
    // TODO: figure out a domain name with associated cert
    // customDomain: {
    //   alternateNames: [api.url],
    //   domainName: 'domain.com',
    // },
  });
  stack.addOutputs({
    StaticEndpoint: staticSite?.url,
    BucketDomainName: staticSite?.bucket?.bucketDomainName,
  });

  return { staticSite };
}
