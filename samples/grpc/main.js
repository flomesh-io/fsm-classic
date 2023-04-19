pipy()
.listen(29001)
.demuxHTTP().to(
  $=>$.muxHTTP({version: 2}).to(
    $=>$.connectTLS({
      trusted: [new crypto.Certificate(pipy.load('ca.crt'))],
      alpn: 'h2'
    }).to(
      $=>$.connect('grpctest.dev:9001')
    )
  )
)