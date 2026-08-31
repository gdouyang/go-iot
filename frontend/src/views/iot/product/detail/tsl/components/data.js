import _ from 'lodash-es'

/** 物模型标识比较：忽略大小写 */
export function sameTslId(a, b) {
  return String(a || '').toLowerCase() === String(b || '').toLowerCase()
}

const defaultPropertiesData = {
  id: null,
  name: null,
  type: null,
  expands: {
    readOnly: null
  },
  description: null
}
const defaultEventsData = {
  id: null,
  name: null,
  type: null,
  expands: {
    level: null
  },
  description: null
}
export function getPropertiesData(data) {
  const d = _.assign({}, defaultPropertiesData, data)
  return d
}

export function getFunctionsData(data) {
  const output = data && data.output ? data.output : {}
  if (output.properties) {
  } else {
    output.properties = []
  }
  const d = _.assign({}, defaultPropertiesData, data)
  d.output = output
  if (d.inputs) {
  } else {
    d.inputs = []
  }
  return d
}

export function getEventsData(data) {
  const d = _.assign({}, defaultEventsData, data)
  return d
}
