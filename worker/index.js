/**
 * Shipman edge Worker.
 *
 *   /api/*  →  proxied to env.BACKEND_URL (the hosted Go backend)
 *   /*      →  served from the built FE assets in Shipman-FE/dist (SPA)
 *
 * Set the backend host once with:
 *   wrangler secret put BACKEND_URL          # e.g. https://shipman-api.fly.dev
 */

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    if (url.pathname.startsWith('/api/')) {
      const backend = env.BACKEND_URL;
      if (!backend) {
        return new Response(
          JSON.stringify({
            error:
              'BACKEND_URL is not configured on the Worker. ' +
              'Run: wrangler secret put BACKEND_URL  (e.g. https://shipman-api.example.com)',
          }),
          { status: 502, headers: { 'content-type': 'application/json' } },
        );
      }

      const target = new URL(url.pathname + url.search, backend.replace(/\/+$/, '') + '/');
      const proxied = new Request(target.toString(), request);
      proxied.headers.set('host', target.host);
      return fetch(proxied);
    }

    return env.ASSETS.fetch(request);
  },
};
