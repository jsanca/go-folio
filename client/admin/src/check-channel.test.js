import { test } from 'vitest'

test('check MessageChannel timing', () => {
  console.log('MC type:', typeof MessageChannel)
  
  if (typeof MessageChannel === 'undefined') {
    console.log('NO MessageChannel in jsdom!')
    return
  }
  
  const ch = new MessageChannel()
  const log = []
  ch.port1.onmessage = () => log.push('mc')
  ch.port2.postMessage(null)
  setTimeout(() => log.push('st0'), 0)
  
  return new Promise(r => {
    setTimeout(() => {
      console.log('order:', JSON.stringify(log))
      r()
    }, 100)
  })
})
