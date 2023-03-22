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
    config = pipy.solve('config.js'),
    logger = config.log.reduce(
      (logger, target) => (
        logger.toHTTP(
          target.url, {
            headers: target.headers,
            batch: target.batch
          }
        )
      ),
      new logging.JSONLogger('http')
    ),

  ) => pipy({
    _reqHead: null,
    _reqTime: 0,
    _reqSize: 0,
    _resHead: null,
    _resTime: 0,
    _resSize: 0,
  })

    .pipeline()
    .handleMessageStart(
      msg => (
        _reqHead = msg.head,
          _reqTime = Date.now()
      )
    )
    .handleData(data => _reqSize += data.size)
    .chain()
    .handleMessageStart(
      msg => (
        _resHead = msg.head,
          _resTime = Date.now()
      )
    )
    .handleData(data => _resSize += data.size)
    .handleMessageEnd(
      () => (
        logger.log({
          req: _reqHead,
          res: _resHead,
          reqSize: _reqSize,
          resSize: _resSize,
          reqTime: _reqTime,
          resTime: _resTime,
          endTime: Date.now(),
        })
      )
    )

)()