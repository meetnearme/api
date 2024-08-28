export default {
  async fetch(req, env) {
    console.log('req: ', req);
    const url = new URL(req.url);
    const hostname = url.hostname;
    const subdomain = hostname.split('.')[0];

    let subdomainValue = null;
    if (subdomain) {
      console.log('subdomain: ', subdomain);
      subdomainValue = await env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);
    }

    console.log('subdomainValue: ', subdomainValue);

    const data =
      req.cf !== undefined
        ? req.cf
        : { error: 'The `cf` object is not available inside the preview.' };

    const headers = new Headers({
      'content-type': 'application/json;charset=UTF-8',
    });

    // TODO: Remove, this is for testing
    headers.set('mnm-subdomain', JSON.stringify({ 'test-key': 'test-value' }));
    if (subdomainValue) {
      headers.set('mnm-subdomain', JSON.stringify(subdomainValue));
    }

    return new Response(JSON.stringify(data, null, 2), {
      headers: headers,
    });
  },
};
