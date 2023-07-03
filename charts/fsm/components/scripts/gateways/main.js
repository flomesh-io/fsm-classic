((
  { config } = pipy.solve('config.js'),
  listeners = {},
) => pipy()

.export('listener', {
  __port: null,
})

.repeat(
  (config.Listeners || []),
  ($, l)=>$.listen(
    (listeners[l.Port] = new ListenerArray, listeners[l.Port].add(l.Port), listeners[l.Port]),
    { ...l, protocol: (l?.Protocol === 'UDP') ? 'udp' : 'tcp' }
  )
  .onStart(
    () => (
      __port = l,
      new Data
    )
  )
  .link('launch')
)

.pipeline('launch')
.branch(
  () => (__port?.Protocol === 'HTTP'), (
    $=>$.chain(config?.Chains?.HTTPRoute || [])
  ),
  () => (__port?.Protocol === 'HTTPS' && __port?.TLS?.TLSModeType === 'Terminate'), (
     $=>$.chain(config?.Chains?.HTTPSRoute || [])
  ),
  () => (__port?.Protocol === 'HTTPS' && __port?.TLS?.TLSModeType === 'Passthrough'), (
    $=>$.chain(config?.Chains?.HTTPSPassthrough || [])
  ),
  () => (__port?.Protocol === 'TLS' && __port?.TLS?.TLSModeType === 'Terminate'), (
    $=>$.chain(config?.Chains?.TLSTerminate || [])
  ),
  () => (__port?.Protocol === 'TCP'), (
    $=>$.chain(config?.Chains?.TCPRoute || [])
  ),
  (
    $=>$.replaceStreamStart(new StreamEnd)
  )
)

)()