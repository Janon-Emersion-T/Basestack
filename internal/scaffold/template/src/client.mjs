// Public HTTP boundary. No runtime configuration or private environment imports.
export function createClient(config, transport = globalThis.fetch) {
  if (!config || config.schemaVersion !== 1 || typeof config.apiURL !== 'string' || Object.keys(config).some(k => !['schemaVersion', 'apiURL'].includes(k))) throw new Error('Invalid public service configuration.');
  let base = config.apiURL;
  if (base) {
    const url = new URL(base);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash || url.pathname !== '/') throw new Error('API URL must be an HTTP(S) origin.');
    base = url.origin;
  }
  let token;
  async function request(path, method = 'GET', body, binary = false) {
    const headers = {};
    if (token) headers.Authorization = `Bearer ${token}`;
    if (body !== undefined && !(body instanceof Blob)) headers['Content-Type'] = 'application/json';
    const response = await transport(base + path, { method, headers, body: body instanceof Blob ? body : body === undefined ? undefined : JSON.stringify(body), credentials: 'omit', redirect: 'error', cache: 'no-store' });
    if (!response.ok) throw new Error(`BaseStack request failed (${response.status}).`);
    if (binary) return response.blob();
    return (await response.json()).data;
  }
  const segment = value => {
    if (typeof value !== 'string' || !/^[a-z0-9][a-z0-9-]{0,63}$/.test(value)) throw new Error('Invalid service identifier.');
    return value;
  };
  return {
    auth: {
      signup: (email, password) => request('/api/auth/signup', 'POST', { email, password }),
      async login(email, password) {
        token = undefined;
        const result = await request('/api/auth/login', 'POST', { email, password });
        if (!result?.sessionIssued || typeof result.session?.token !== 'string') throw new Error('Server does not support sessions.');
        token = result.session.token;
        return result.user;
      },
      current: async () => (await request('/api/auth/me')).user,
      async logout() { try { return await request('/api/auth/logout', 'POST'); } finally { token = undefined; } },
      verifyEmail: token => request('/api/auth/verify-email', 'POST', { token }),
      requestVerification: email => request('/api/auth/verify-email/request', 'POST', { email }),
      forgotPassword: email => request('/api/auth/password/forgot', 'POST', { email }),
      resetPassword: (token, password) => request('/api/auth/password/reset', 'POST', { token, password }),
    },
    functions: { run: (name, input = {}) => request(`/api/functions/${segment(name)}`, 'POST', input) },
    storage: {
      list: bucket => request(`/api/storage/${segment(bucket)}`),
      upload: (bucket, blob) => request(`/api/storage/${segment(bucket)}`, 'POST', blob),
      download: (bucket, id) => request(`/api/storage/${segment(bucket)}/${segment(id)}`, 'GET', undefined, true),
      delete: (bucket, id) => request(`/api/storage/${segment(bucket)}/${segment(id)}`, 'DELETE'),
    },
  };
}
