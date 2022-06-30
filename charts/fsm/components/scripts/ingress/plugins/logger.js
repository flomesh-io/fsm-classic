(config =>

  pipy({
    _enabled: config.enabled,
    _logURL: config.enabled && new URL(config.logURL),
    _request: null,
    _requestTime: 0,
    _responseTime: 0,
    _responseContentType: '',
    _instanceName: os.env.PIPY_SERVICE_NAME,

    _CONTENT_TYPES: {
      '': true,
      'text/plain': true,
      'application/json': true,
      'application/xml': true,
      'multipart/form-data': true,
    },

  })

  .import({
      __apiID: 'router',
      __ingressID: 'router',
      __projectID: 'router'
    })

  .export('logger', {
      __logInfo: {},
    })

  .pipeline('request')
    .link('req', () => _enabled === true, 'bypass')
    .fork('log-request')

  .pipeline('response')
    .link('resp', () => _enabled === true, 'bypass')

  .pipeline('req')
    .fork('log-request')

  .pipeline('resp')
    .fork('log-response')

  .pipeline('log-request')
    .handleMessageStart(
      () => _requestTime = Date.now()
    )
    .decompressHTTP()
    .handleMessage(
      '1024k',
      msg => (
        _request = msg
      )
    )

  .pipeline('log-response')
    .handleMessageStart(
      msg => (
        _responseTime = Date.now(),
        _responseContentType = (msg.head.headers && msg.head.headers['content-type']) || ''
      )
    )
    .decompressHTTP()
    .replaceMessage(
      '1024k',
      msg => (
        new Message(
          JSON.encode({
            req: {
              ..._request.head,
              body: _request.body.toString(),
            },
            res: {
              ...msg.head,
              body: Boolean(_CONTENT_TYPES[_responseContentType]) ? msg.body.toString() : '',
            },
            x_parameters: { aid: __apiID, igid: __ingressID, pid: __projectID },
            instanceName: _instanceName,
            reqTime: _requestTime,
            resTime: _responseTime,
            endTime: Date.now(),
            reqSize: _request.body.size,
            resSize: msg.body.size,
            remoteAddr: __inbound?.remoteAddress,
            remotePort: __inbound?.remotePort,
            localAddr: __inbound?.localAddress,
            localPort: __inbound?.localPort,
            ...__logInfo,
          }).push('\n')
        )
      )
    )
    .merge('log-send', () => '')

  .pipeline('log-send')
    .pack(
      1000,
      {
        timeout: 5,
      }
    )
    .replaceMessageStart(
      () => new MessageStart({
        method: 'POST',
        path: _logURL.path,
        headers: {
          'Host': _logURL.host,
          'Authorization': config.Authorization,
          'Content-Type': 'application/json',
        }
      })
    )
    .encodeHTTPRequest()

    .connect(
      () => _logURL.host,
      {
        bufferLimit: '8m',
      }
    )

  .pipeline('bypass')

)(JSON.decode(pipy.load('config/logger.json')))
