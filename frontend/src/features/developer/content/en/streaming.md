# Streaming responses

The Chat Completions and Responses endpoints support the stream request field. The server returns incremental events using Server-Sent Events.

~~~json
{
  "model": "your-model",
  "stream": true,
  "messages": [{"role": "user", "content": "Write a short poem"}]
}
~~~

## Boundaries to handle

- Read the response incrementally; one network chunk is not guaranteed to contain one complete event.
- Keep a buffer across chunks and split events on blank lines.
- Abort the request and release the reader when the user cancels.
- Treat timeouts, network failures, and malformed streams separately.

Live Playground mode renders streaming text and provides a stop control. Stream content is retained only in the current page state.
