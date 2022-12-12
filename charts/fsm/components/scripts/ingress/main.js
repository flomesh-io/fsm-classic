/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

(({
    config,
    certificates,
    issuingCAs
  } = pipy.solve('config.js')) =>

  pipy({
    _passthroughTarget: undefined,
  })
  .export('main', {
    __route: undefined,
    __isTLS: false,
  })

  .listen(config.listen)
    .link('inbound-http')

  .listen(config.listenTLS)
    .link(
      'passthrough', () => config?.sslPassthrough?.enabled === true,
      'inbound-tls'
    )

  .pipeline('inbound-tls')
    .onStart(() => void(__isTLS = true))
    .acceptTLS({
      certificate: config.listenTLS && config.certificate && config.certificate.cert && config.certificate.key ? {
        cert: new crypto.CertificateChain(config.certificate.cert),
        key: new crypto.PrivateKey(config.certificate.key),
      } : undefined,
    }).to('inbound-http')

  .pipeline('passthrough')
    .handleTLSClientHello(
      hello => (
        _passthroughTarget = hello.serverNames ? hello.serverNames[0] || '' : ''
      )
    )
    .branch(
      () => Boolean(_passthroughTarget), (
        $=>$.connect(() => `${_passthroughTarget}:${config.sslPassthrough.upstreamPort}`)
      ),
      (
        $=>$.replaceStreamStart(new StreamEnd)
      )
    )

  .pipeline('inbound-http')
    .demuxHTTP().to(
      $=>$.chain(config.plugins)
    )
)()
