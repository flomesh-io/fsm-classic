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

(config =>

  pipy({
    _certificates: config.certificates && Object.fromEntries(
      Object.entries(config.certificates).map(
        ([k, v]) => [
          k, {
            cert: new crypto.CertificateChain(v.cert),
            key: new crypto.PrivateKey(v.key),
          }
        ]
      )
    ),
  })

  .export('main', {
      __turnDown: false,
      __isTLS: false,
    })

  .listen(config.listen)
    .link('tls-offloaded')

  .listen(config.tlsport)
    .handleStreamStart(
      () => __isTLS = true
    )
    .acceptTLS('tls-offloaded', {
      certificate: sni => sni && Object.entries(_certificates).find(([k, v]) => new RegExp(k).test(sni))?.[1]
    })


  .pipeline('tls-offloaded')
    .use(config.plugins, 'session')
    .demuxHTTP('request')

    .pipeline('request')
    .use(
      config.plugins,
      'request',
      'response',
      () => __turnDown
    )

)(JSON.decode(pipy.load('config/main.json')))
