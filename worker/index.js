addEventListener('fetch', (event) => {
  event.respondWith(handleRequest(event.request, event));
});

async function handleRequest(request, event) {
  const url = new URL(request.url);
  const hostname = url.hostname;
  const parts = hostname.split('.');
  const subdomain = parts.slice(0, -2).join('.');

  let subdomainValue = null;

  if (subdomain) {
    subdomainValue = await event.env.MNM_SUBDOMAIN_KV_NAMESPACE.get(subdomain);

    // Add custom headers to the original request
    if (subdomainValue) {
      request.headers.set('X-Mnm-Subdomain-Value', subdomainValue);
    }
  }

  // Pass through the modified request
  return fetch(request);
}
