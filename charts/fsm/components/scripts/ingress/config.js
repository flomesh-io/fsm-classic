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
    config = JSON.decode(pipy.load('config/main.json')),
    ingress = pipy.solve('ingress.js'),
    global
  ) => (
    global = {
      mapIssuingCA: {},
      issuingCAs: [],
      certificates: {}
    },

    global.addIssuingCA = ca => (
      (md5 => (
        md5 = '' + algo.hash(ca),
        !global.mapIssuingCA[md5] && (
          global.issuingCAs.push(new crypto.Certificate(ca)),
            global.mapIssuingCA[md5] = true
        )
      ))()
    ),

    global.prepareQuote = (str, delimiter) => (
      ((str + '').replace(new RegExp('[.\\\\+*?\\[\\^\\]$(){}=!<>|:\\' + (delimiter || '') + '-]', 'g'), '\\$&'))()
    ),

    global.globStringToRegex = (str) => (
      (new RegExp(global.prepareQuote(str).replace(new RegExp('\\\*', 'g'), '.*').replace(new RegExp('\\\?', 'g'), '.'), 'g'))()
    ),

    global.wildcardDomain = new RegExp("^(\\*\\.)?([\\w-]+\\.)+[\\w-]+$"),

    global.issuingCAs && (
      Object.values(ingress.certificates).forEach(
        (v) => (
          v?.ca && (
            global.addIssuingCA(v.ca)
          )
        )
      )
    ),

    config?.certificate?.ca && (
      global.addIssuingCA(config.certificate.ca)
    ),

    global.certificates = (
      Object.fromEntries(
        Object.entries(ingress.certificates).map(
          ([k, v]) =>(
            (() => (
              [k, {
                cert: new crypto.CertificateChain(v.cert),
                key: new crypto.PrivateKey(v.key),
                regex: global.wildcardDomain.test(k) ? global.globStringToRegex(k) : undefined
              }]
            ))()
          )
        )
      )
    ),
    global.config = config,

    global
  )
)()