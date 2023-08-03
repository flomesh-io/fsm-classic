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
  (
    { isDebugEnabled } = pipy.solve('config.js'),

    {
      namespace,
      kind,
      name,
      pod,
      toInt63,
    } = pipy.solve('lib/utils.js'),

    tracingAddress = os.env.TRACING_ADDRESS,
    tracingEndpoint = (os.env.TRACING_ENDPOINT || '/api/v2/spans'),
    tracingLimitedID = os.env.TRACING_SAMPLED_FRACTION && (os.env.TRACING_SAMPLED_FRACTION * Math.pow(2, 63)),
    logZipkin = tracingAddress && new logging.JSONLogger('zipkin').toHTTP('http://' + tracingAddress + tracingEndpoint, {
      batch: {
        timeout: 1,
        interval: 1,
        prefix: '[',
        postfix: ']',
        separator: ','
      },
      headers: {
        'Host': tracingAddress,
        'Content-Type': 'application/json',
      }
    }).log,

  ) => (
    {
      tracingEnabled: Boolean(logZipkin),

      initTracingHeaders: (headers, proto) => (
        (
          sampled = true,
          uuid = algo.uuid(),
          id = uuid.substring(0, 18).replaceAll('-', ''),
        ) => (
          proto && (headers['x-forwarded-proto'] = proto),
          headers['x-b3-spanid'] && (
            (headers['x-b3-parentspanid'] = headers['x-b3-spanid']) && (headers['x-b3-spanid'] = id)
          ),
          !headers['x-b3-traceid'] && (
            (headers['x-b3-traceid'] = id) && (headers['x-b3-spanid'] = id)
          ),
          headers['x-b3-sampled'] && (
            sampled = (headers['x-b3-sampled'] === '1'), true
          ) || (
            (sampled = (!tracingLimitedID || toInt63(headers['x-b3-traceid']) < tracingLimitedID)) ? (headers['x-b3-sampled'] = '1') : (headers['x-b3-sampled'] = '0')
          ),
          !headers['x-request-id'] && (
            headers['x-request-id'] = uuid
          ),
          headers['fgw-stats-namespace'] = namespace,
          headers['fgw-stats-kind'] = kind,
          headers['fgw-stats-name'] = name,
          headers['fgw-stats-pod'] = pod,
          sampled
        )
      )(),

      makeZipKinData: (msg, headers, serviceName) => (
        (data) => (
          data = {
            'traceId': headers?.['x-b3-traceid'] && headers['x-b3-traceid'].toString(),
            'id': headers?.['x-b3-spanid'] && headers['x-b3-spanid'].toString(),
            'name': headers?.host,
            'timestamp': Date.now() * 1000,
            'localEndpoint': {
              'port': 0,
              'ipv4': os.env.POD_IP || '',
              'serviceName': name,
            },
            'tags': {
              'component': 'proxy',
              'http.url': headers?.['x-forwarded-proto'] + '://' + headers?.host + msg?.head?.path,
              'http.method': msg?.head?.method,
              'node_id': os.env.POD_UID || '',
              'http.protocol': msg?.head?.protocol,
              'guid:x-request-id': headers?.['x-request-id'],
              'user_agent': headers?.['user-agent'],
              'upstream_service': serviceName
            },
            'annotations': []
          },
          headers['x-b3-parentspanid'] && (data['parentId'] = headers['x-b3-parentspanid']),
          data['kind'] = 'fgw',
          data['shared'] = false,
          data.tags['request_size'] = '0',
          data.tags['response_size'] = '0',
          data.tags['http.status_code'] = '502',
          data.tags['peer.address'] = '',
          data['duration'] = 0,
          data
        )
      )(),

      saveTracing: (zipkinData, messageHead, bytesStruct, target) => (
        zipkinData && (
          zipkinData.tags['peer.address'] = target,
          zipkinData.tags['http.status_code'] = messageHead.status?.toString?.(),
          zipkinData.tags['request_size'] = bytesStruct.requestSize.toString(),
          zipkinData.tags['response_size'] = bytesStruct.responseSize.toString(),
          zipkinData['duration'] = Date.now() * 1000 - zipkinData['timestamp'],
          logZipkin(zipkinData),
          isDebugEnabled && console.log('[tracing] json:', zipkinData)
        )
      ),
    }
  )

)()