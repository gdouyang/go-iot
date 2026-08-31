export function newTcpAddObj() {
  return {
    id: null,
    name: '',
    type: 'TCP_SERVER',
    productId: null,
    configuration: {
      useTLS: false,
      certificate: null,
      host: '',
      delimeter: {
        type: 'NONE', // Delimited, FixLength
        splitFunc: null,
        length: null
      }
    }
  }
}

export function parserType(type) {
  if (type === 'Delimited') {
    return '分隔符'
  } else if (type === 'FixLength') {
    return '固定长度'
  } else if (type === 'SplitFunc') {
    return '自定义脚本'
  } else {
    return '不处理'
  }
}

export function newMqttAddObj() {
  return {
    id: null,
    name: '',
    type: 'MQTT_BROKER',
    productId: null,
    configuration: {
      useTLS: false,
      certificate: null,
      host: '',
      port: null
    }
  }
}

export function newWebSocketAddObj() {
  return {
    id: null,
    name: '',
    type: 'WEBSOCKET_SERVER',
    productId: null,
    configuration: {
      useTLS: false,
      certificate: null,
      host: '',
      routers: [{ url: '' }]
    }
  }
}

export function newHttpAddObj() {
  return {
    id: null,
    name: '',
    type: 'HTTP_SERVER',
    productId: null,
    configuration: {
      useTLS: false,
      certificate: null,
      host: '',
      routers: [{ url: '' }]
    }
  }
}

export function newCoAPAddObj() {
  return {
    id: null,
    name: '',
    type: 'COAP_SERVER',
    productId: null,
    configuration: {
      useTLS: false,
      certificate: null,
      host: '',
      routers: [{ url: '' }]
    }
  }
}
