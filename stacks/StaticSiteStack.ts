import { Api, StaticSite, use } from 'sst/constructs';
import { ApiStack } from './ApiStack';
const util = require('node:util');

// TODO: pass `api` created from that stack in to get `api.url` here for `alternateNames`
export function StaticSiteStack({ stack }: StackContext) {
  const { api } = use(ApiStack);
  console.log('~api.url', util.inspect(api.url));
  const staticSite = new StaticSite(stack, 'frontend', {
    path: 'static',
    customDomain: {
      alternateNames: [api.url],
    },
  });

  return { staticSite };
}
