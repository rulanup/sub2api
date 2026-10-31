# Authentication

Gateway endpoints authenticate with an API key. Send it in the Authorization header:

~~~http
Authorization: Bearer <your-api-key>
~~~

## Security rules

- Never commit API keys to Git, public issues, or frontend source code.
- Do not put keys in URL query parameters.
- Create separate keys for separate applications so usage and revocation stay manageable.
- Use HTTPS for production traffic.

## Browser debugging

The portal's Live Playground keeps the key in page memory only. Refreshing, closing the page, or clearing the field removes it. The key is not written to localStorage, IndexedDB, cookies, or the URL.

Browser requests are also subject to the target service's CORS policy. Sub2API does not add a default public proxy that carries user credentials.
