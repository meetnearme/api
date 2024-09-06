export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const hostname = url.hostname;
    const parts = hostname.split('.');
    const subdomain = parts.slice(0, -2).join('.');
    let subdomainValue = null;

    // Clone the original request
    const newRequest = new Request(req);

    if (subdomain) {
      subdomainValue = await env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);

      // Add custom headers
      if (subdomainValue) {
        newRequest.headers.set('X-Mnm-Subdomain-Value', subdomainValue);
      }
    }

    const response = await fetch(newRequest);

    // Create a new response with the same body, status, and headers
    const newResponse = new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers,
    });

    // Add the custom header to the new response if it exists
    if (subdomainValue) {
      newResponse.headers.set('X-Mnm-Subdomain-Value', subdomainValue);
    }

    // Return the new Response
    return newResponse;
  },
};
