export function newScene() {
  return {
    name: '',
    triggerType: '', // device, timer
    cron: '', // type == 'timer'时才有
    state: 'stopped',
    productId: undefined,
    deviceIds: [],
    trigger: {
      filterType: undefined,
      shakeLimit: {
        enabled: false,
        time: 1,
        threshold: 1,
        alarmFirst: true
      },
      filters: []
    }
  }
}

export function newFilter() {
  return {
    logic: undefined,
    key: undefined,
    operator: undefined,
    value: undefined
  }
}
