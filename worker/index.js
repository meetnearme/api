export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const hostname = url.hostname;
    const parts = hostname.split('.');
    const subdomain = parts.slice(0, -2).join('.');

    let subdomainValue = null;

    // Clone the original request
    let newRequest = new Request(req);

    if (subdomain) {
      subdomainValue = await env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);

      // Add custom headers
      if (subdomainValue) {
        newRequest.headers.set('X-Mnm-Subdomain-Value', subdomainValue);
      }
    }

    // Return the modified request without making a fetch call
    return newRequest;
  },
};
