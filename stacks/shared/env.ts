export default {
  MEETNEARME_TEST_SECRET: process.env.MEETNEARME_TEST_SECRET,
  ZENROWS_API_KEY: process.env.ZENROWS_API_KEY,
  // STATIC_BASE_URL is a special case because the value comes from
  // `sst deploy` at runtime and then gets set as an environment variable
  STATIC_BASE_URL: process.env.STATIC_BASE_URL ?? staticSite.url,
  // ROUTE53_HOSTED_ZONE_ID omitted because it's only used in deployment
  // ROUTE53_HOSTED_ZONE_NAME omitted because it's only used in deployment
};
