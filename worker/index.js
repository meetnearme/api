/**
 * Shared logic for subdomain extraction and mnmOptions formatting
 * Used by both production and development workers
 */

/**
 * Extracts subdomain from hostname
 * @param {string} hostname - The hostname from the request URL
 * @returns {string|null} - The subdomain or null if no subdomain
 */
export function extractSubdomain(hostname, localDev = false) {
  const parts = hostname.split('.');
  if (parts.length === 1) {
    return null;
  }
  if (localDev) {
    // For local dev (e.g., "subdomain.localhost:8000"), just take the first part
    return parts[0];
  }
  // Subdomain is everything except the last 2 parts (e.g., "subdomain" from "subdomain.example.com")
  const subdomain = parts.slice(0, -2).join('.');

  return subdomain || null;
}

/**
 * Normalizes mnmOptions to the expected format
 * Handles legacy format where value is just userId (no key=value pairs)
 * @param {string|null} mnmOptions - Raw mnmOptions from KV store
 * @returns {string|null} - Normalized mnmOptions string
 */
export function normalizeMnmOptions(mnmOptions) {
  if (!mnmOptions) {
    return null;
  }

  // legacy implementation of this had only a string value for
  // subdomain, but now we store it as semicolon-delimited key-value
  // pairs with = as the separator (JSON isn't good as a header value)
  if (!mnmOptions.includes('=')) {
    return `userId=${mnmOptions}`;
  }

  return mnmOptions;
}

/**
 * Processes subdomain and fetches mnmOptions from KV
 * @param {string} hostname - The hostname from the request URL
 * @param {Function} fetchMnmOptions - Function to fetch mnmOptions from KV (returns Promise<string|null>)
 * @returns {Promise<string|null>} - Normalized mnmOptions or null
 */
export async function getMnmOptionsFromSubdomain(
  hostname,
  localDev,
  fetchMnmOptions,
) {
  const subdomain = extractSubdomain(hostname, localDev);
  if (!subdomain) {
    return null;
  }

  const rawMnmOptions = await fetchMnmOptions(subdomain);
  return normalizeMnmOptions(rawMnmOptions);
}

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const hostname = url.hostname;

    // Clone the original request
    const newRequest = new Request(req);

    // Fetch mnmOptions using KV binding
    const mnmOptions = await getMnmOptionsFromSubdomain(
      hostname,
      false,
      async (subdomain) => {
        return await env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);
      },
    );

    // Add custom headers
    if (mnmOptions) {
      newRequest.headers.set('X-Mnm-Options', JSON.stringify(mnmOptions));
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
