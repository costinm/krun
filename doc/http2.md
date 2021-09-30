Golang HTTP/2 has 2 small and one big problem when attempting to use it for
streams.

1. Server API, the server side of the stream can't be closed - there is no
   'Close' method. Returning from the handle() method would eventually close -
   but only after the input stream has been consumed, so there is no way for the
   server to initate a FIN.
2. For client API, there is a deadline when server returns an error and client
   reader is a stream. The code will attempt to cancel the thread copying the
   input - but the input is blocked on a Read(). For normal POST it's fine, but
   if you pass a Reader with a read that blocks - like a pipe - this never
   completes. Workaround is possible but ugly.
3. A LOT of overhead, due to the expectations of implementing http semantics.

Options:
- use the framer directly - similar with gRPC
- use gRPC - which has it's own framing and library
- use the Rust http stack, wrapped as a native library
