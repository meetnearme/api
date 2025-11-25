/**
 * Local development worker that proxies to localhost
 * Uses Cloudflare API to fetch KV values instead of KV bindings
 * Reads CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID from environment
 */

/* eslint-disable no-console */
import { getMnmOptionsFromSubdomain, extractSubdomain } from './index.js';

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const hostname = url.hostname;
    console.log('Original hostname:', hostname);

    // Extract subdomain for forwarding (preserve it in the localhost URL)
    const subdomain = extractSubdomain(hostname, true);
    console.log('Extracted subdomain:', subdomain);

    // Fetch mnmOptions using Cloudflare API
    let mnmOptions = null;
    if (subdomain) {
      mnmOptions = await getMnmOptionsFromSubdomain(
        hostname,
        true,
        async (subdomain) => {
          const apiToken = env.CLOUDFLARE_API_TOKEN;
          const accountId = env.CLOUDFLARE_ACCOUNT_ID;
          const kvNamespaceId = env.CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID;

          if (!apiToken || !accountId) {
            console.error('No API token or account ID');
            return null;
          }

          const apiBaseUrl =
            env.CLOUDFLARE_API_BASE_URL || 'https://api.cloudflare.com';
          try {
            // Fetch from KV using Cloudflare API
            // Use the same URL pattern as the Go service
            // Note: GET requests to KV API don't need Content-Type header
            const kvResponse = await fetch(
              `${apiBaseUrl}/client/v4/accounts/${accountId}/storage/kv/namespaces/${kvNamespaceId}/values/${subdomain}`,
              {
                method: 'GET',
                headers: {
                  Authorization: `Bearer ${apiToken}`,
                },
              },
            );
            const res = await kvResponse.text();
            if (kvResponse.ok) {
              return res;
            } else {
              console.error('Error fetching from KV:', kvResponse.statusText);
            }
          } catch (error) {
            console.error('Error fetching from KV:', error);
            // Continue without mnmOptions if KV fetch fails
          }

          return null;
        },
      );
    }

    // Clone the original request and modify URL to point to localhost
    // Preserve the subdomain in the forwarded URL (e.g., subdomain.localhost:8000)
    // Note: Cloudflare Workers fetch doesn't allow "localhost", so we use 127.0.0.1
    // but we still set the Host header to preserve the subdomain for the Go app
    const localHostname = subdomain ? `${subdomain}.localhost` : null;
    const localUrl = new URL(`http://127.0.0.1:8000`);
    localUrl.pathname = url.pathname;
    localUrl.search = url.search;
    localUrl.hash = url.hash;

    console.log('Forwarding to:', localUrl.toString());

    // Create new headers (clone original headers)
    const newHeaders = new Headers(req.headers);

    // set X-Original-Host as a backup since Host header may be overridden by HTTP client
    if (localHostname) {
      console.log('X-Original-Host header:', localHostname);
      newHeaders.set('X-Original-Host', localHostname);
    }

    // Add X-Mnm-Options header if we have it
    if (mnmOptions) {
      newHeaders.set('X-Mnm-Options', JSON.stringify(mnmOptions));
    }

    // Create new request with modified URL and headers
    // Note: We need to clone the request body properly
    const requestInit = {
      method: req.method,
      headers: newHeaders,
    };

    // Only include body if the request has one (GET requests don't have bodies)
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      requestInit.body = req.body;
    }

    const newRequest = new Request(localUrl.toString(), requestInit);

    // Forward request to local API
    let response;
    try {
      response = await fetch(newRequest);
    } catch (error) {
      console.error('Error forwarding request to local API:', error);
      console.error('Attempted URL:', localUrl.toString());
      return new Response(
        JSON.stringify({
          error: 'Failed to forward request to local API',
          message: error.message,
          attemptedUrl: localUrl.toString(),
        }),
        {
          status: 502,
          headers: { 'Content-Type': 'application/json' },
        },
      );
    }

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
