export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const hostname = url.hostname;
    const parts = hostname.split('.');
    const subdomain = parts.slice(0, -2).join('.');
    let mnmOptions = null;

    // Clone the original request
    const newRequest = new Request(req);

    if (subdomain) {
      // this has *_SUBDOMIN_* because it was initially envisioned as
      // containing only a subdomain mapping, but now it contains
      // other user config options as well
      mnmOptions = await env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);

      // legacy implementation of this had only a string value for
      // subdomain, but now we store it as semicolon-delimited key-value
      // pairs with = as the separator (JSON isn't good as a header value)
      if (!mnmOptions?.includes('=')) {
        mnmOptions = `userId=${mnmOptions}`;
      }
      // Add custom headers
      if (mnmOptions) {
        newRequest.headers.set('X-Mnm-Options', JSON.stringify(mnmOptions));
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
    if (mnmOptions) {
      newResponse.headers.set('X-Mnm-Options', mnmOptions);
    }

    // Return the new Response
    return newResponse;
  },
};
