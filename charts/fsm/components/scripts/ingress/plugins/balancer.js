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

(
  (config = JSON.decode(pipy.load('config/balancer.json')),
  global
) => (
  global = {
    mapIssuingCA: {},
    listIssuingCA: [],
    services: {}
  },

  global.addIssuingCA = ca => (
    (md5 => (
      md5 = '' + algo.hash(ca),
      !global.mapIssuingCA[md5] && (
        global.listIssuingCA.push(new crypto.Certificate(ca)),
          global.mapIssuingCA[md5] = true
      )
    ))()
  ),

  global.services = Object.fromEntries(
    Object.entries(config.services).map(
      ([k, v]) => (
        ((balancer => (
            balancer = new algo[v.balancer ? v.balancer : 'RoundRobinLoadBalancer'](v.targets),
            v?.proxySslCert?.ca && (
              global.addIssuingCA(v.proxySslCert.ca)
            ),
            [k, {
              balancer,
              cache: v.sticky && new algo.Cache(
                () => balancer.select()
              ),
              proxySslName: v?.proxySslName || null,
              proxySslVerify: v?.proxySslVerify || false,
              cert: v?.proxySslCert?.cert,
              key: v?.proxySslCert?.key
            }]
          ))()
        )
      )
    )
  ),

  pipy({
    _requestCounter: new stats.Counter('http_requests_count', ['method', 'status', "host", "path"]),
    _requestLatency: new stats.Histogram('http_request_latency', [1, 2, 5, 7, 10, 15, 20, 25, 30, 40, 50, 60, 70,
      80, 90, 100, 200, 300, 400, 500, 1000,
      2000, 5000, 10000, 30000, 60000, Number.POSITIVE_INFINITY]),
    _reqHead: null,
    _resHead: null,
    _reqTime: 0,
    _service: null,
    _serviceSNI: null,
    _serviceVerify: null,
    _serviceCertChain: null,
    _servicePrivateKey: null,
    _tlsOptions: null,
    _serviceCache: null,
    _target: '',
    _targetCache: null,

    _g: {
      connectionID: 0,
    },

    _connectionPool: new algo.ResourcePool(
      () => ++_g.connectionID
    ),

    _selectKey: null,
    _select: (service, key) => (
      service.cache && key ? (
        service.cache.get(key)
      ) : (
        service.balancer.select()
      )
    ),
  })

  .import({
      __turnDown: 'main',
      __service: 'router',
    })

  .pipeline('session')
    .handleStreamStart(
      () => (
        _serviceCache = new algo.Cache(
          // k is a balancer, v is a target
          (k) => _select(k, _selectKey),
          (k, v) => k.balancer.deselect(v),
        ),
        _targetCache = new algo.Cache(
          // k is a target, v is a connection ID
          (k) => _connectionPool.allocate(k),
          (k, v) => _connectionPool.free(v),
        )
      )
    )
    .handleStreamEnd(
      () => (
        _targetCache.clear(),
        _serviceCache.clear()
      )
    )

  .pipeline('request')
    .handleMessageStart(
      (msg) => (
        _selectKey = msg.head?.headers?.['authorization'],
        _service = global.services[__service],
        _service && (
          _serviceSNI = _service?.proxySslName,
          _serviceVerify = _service?.proxySslVerify,
          _serviceCertChain = _service?.cert,
          _servicePrivateKey = _service?.key,
          _target = _serviceCache.get(_service)
        ),
        _serviceCertChain && _servicePrivateKey && (
          _tlsOptions = {
            certificate: () => ({
              cert: new crypto.Certificate(_serviceCertChain),
              key: new crypto.PrivateKey(_servicePrivateKey),
            }),
            trusted: global.listIssuingCA,
            verify: (ok, cert) => (
              !_serviceVerify && (ok = true),
                ok
            )
          },
          _serviceSNI && (
            _tlsOptions['sni'] = _serviceSNI
          )
        ),
        _target && (msg.head.headers['host'] = _target.split(":")[0]),
        _target && (__turnDown = true)
      )
    )
    .link(
      'forward', () => Boolean(_target) && !Boolean(_tlsOptions),
      'forward-tls', () => Boolean(_target) && Boolean(_tlsOptions),
      ''
    )

  .pipeline('forward')
    .handleMessageStart(
      msg => (
        _reqHead = msg.head,
        _reqTime = Date.now()
      )
    )
    .muxHTTP(
      'connection',
      () => _targetCache.get(_target)
    )
    .handleMessageStart(
      msg => (
        _resHead = msg.head,
        _requestCounter.withLabels(_reqHead.method, _resHead.status, _reqHead.headers.host, _reqHead.path).increase(),
        _requestLatency.observe(Date.now() - _reqTime)
      )
    )

  .pipeline('connection')
    .handleMessage(
      msg => (console.log('Ingress connection: ' + _target))
    )
    .connect(
      () => _target
    )

  .pipeline('forward-tls')
    .handleMessageStart(
      msg => (
        _reqHead = msg.head,
        _reqTime = Date.now()
      )
    )
    .muxHTTP(
      'connection-tls',
      () => _targetCache.get(_target)
    )
    .handleMessageStart(
      msg => (
        _resHead = msg.head,
        _requestCounter.withLabels(_reqHead.method, _resHead.status, _reqHead.headers.host, _reqHead.path).increase(),
        _requestLatency.observe(Date.now() - _reqTime)
      )
    )

  .pipeline('connection-tls')
    .handleMessage(
      msg => (console.log('Ingress TLS connection: ' + _target))
    )
    .connectTLS(() => _tlsOptions)
    .to($ => $
      .connect(() => _target)
    )

))()
