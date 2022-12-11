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
((
    ingress = pipy.solve('ingress.js'),
    upstreamMapIssuingCA = {},
    upstreamIssuingCAs = [],
    addUpstreamIssuingCA = (ca) => (
      (md5 => (
        md5 = '' + algo.hash(ca),
        !upstreamMapIssuingCA[md5] && (
          upstreamIssuingCAs.push(new crypto.Certificate(ca)),
          upstreamMapIssuingCA[md5] = true
        )
      ))()
    ),
    balancers = {
      'round-robin': algo.RoundRobinLoadBalancer,
      'least-work': algo.LeastWorkLoadBalancer,
      'hashing': algo.HashingLoadBalancer,
    },
    services = (
      Object.fromEntries(
        Object.entries(ingress.services).map(
          ([k, v]) =>(
            targets = v?.upstream?.endpoints?.map(ep => `${ep.ip}:${ep.port}`),
            v?.upstream?.sslCert?.ca && (
              addUpstreamIssuingCA(v.upstream.sslCert.ca)
            ),
            [k, {
              balancer: new balancers[v.balancer](targets || []),
              upstreamSSLName: v?.upstream?.sslName || null,
              upstreamSSLVerify: v?.upstream?.sslVerify || false,
              cert: v?.upstream?.sslCert?.cert,
              key: v?.upstream?.sslCert?.key
            }]
          )
        )
      )
    ),

  ) => pipy({
    _target: undefined,
    _service: null,
    _serviceSNI: null,
    _serviceVerify: null,
    _serviceCertChain: null,
    _servicePrivateKey: null,
    _connectTLS: null,
  })

    .import({
      __route: 'main',
    })

    .pipeline()
    .handleMessageStart(
      (msg) => (
        _service = services[__route],
        _service && (
          _serviceSNI = _service?.upstreamSSLName,
          _serviceVerify = _service?.upstreamSSLVerify,
          _serviceCertChain = _service?.cert,
          _servicePrivateKey = _service?.key,
          _target = _service?.balancer?.next?.()
        ),
        _connectTLS = _serviceCertChain && _servicePrivateKey
      )
    )
    .branch(
      () => Boolean(_target) && !Boolean(_connectTLS), (
        $=>$.muxHTTP(() => _target).to(
          $=>$.connect(() => _target)
        )
      ), () => Boolean(_target) && Boolean(_connectTLS), (
        $=>$.muxHTTP(() => _target).to(
          $=>$.connectTLS({
            certificate: () => ({
              cert: new crypto.Certificate(_serviceCertChain),
              key: new crypto.PrivateKey(_servicePrivateKey),
            }),
            trusted: upstreamIssuingCAs,
            sni: () => _serviceSNI || undefined,
            verify: (ok, cert) => (
              !_serviceVerify && (ok = true),
                ok
            )
          }).to(
            $=>$.connect(() => _target)
          )
        )
      ), (
        $=>$.chain()
      )
    )
)()