((
  config = pipy.solve('config.js'),
  probeScheme = config?.Spec?.Probes?.LivenessProbes?.[0]?.httpGet?.scheme,
  _ = pipy.exec(['sh', '-c', 'while [ "$(ip addr show dev ' + (os.env.CNI_BRIDGE_ETH || 'cni0') + ' 2>&1 | grep inet > /dev/null; echo $?)" -ne 0 ]; do sleep 0.1; done;']),
  bridgeIP = pipy.exec('ip addr show dev ' + (os.env.CNI_BRIDGE_ETH || 'cni0')).toString().split('\n').find(s => s.trim().startsWith('inet'))?.trim?.()?.split?.(' ')?.[1]?.split?.('/')?.[0] || '0.0.0.0',
) => pipy()

.branch(
  Boolean(config?.Inbound?.TrafficMatches), (
    $=>$
    .listen(bridgeIP + ':15003', { transparent: true })
    .onStart(() => new Data)
    .use('modules/inbound-main.js')
  )
)

.branch(
  Boolean(config?.Outbound || config?.Spec?.Traffic?.EnableEgress), (
    $=>$
    .listen(bridgeIP + ':15001', { transparent: true })
    .onStart(() => new Data)
    .use('modules/outbound-main.js')
  )
)

.listen(probeScheme ? 15901 : 0)
.use('probes.js', 'liveness')

.listen(probeScheme ? 15902 : 0)
.use('probes.js', 'readiness')

.listen(probeScheme ? 15903 : 0)
.use('probes.js', 'startup')

.listen(15010)
.use('stats.js', 'prometheus')

.listen(':::15000')
.use('stats.js', 'ecnet-stats')

//
// Local DNS server
//
.branch(
  Boolean(config?.Spec?.LocalDNSProxy), (
    $=>$
    .listen(bridgeIP + ':15053', { protocol: 'udp', transparent: true } )
    .chain(['dns-main.js'])
  )
)

)()
