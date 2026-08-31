export function newNotifierAction() {
  return {
    executor: undefined,
    configuration: {
      notifyType: undefined,
      notifierId: undefined,
      templateId: undefined
    }
  }
}

export function newDeviceMessageAction() {
  return {
    executor: undefined,
    configuration: {
      deviceId: undefined,
      productId: undefined,
      messageType: undefined,
      properties: undefined,
      functionId: undefined,
      data: {}
    }
  }
}

export function WRITE_PROPERTY() {
  return {
    messageType: undefined,
    properties: undefined
  }
}
export function INVOKE_FUNCTION() {
  return {
    messageType: undefined,
    functionId: undefined,
    data: {}
  }
}

export function newEmtpyAction() {
  return {
    executor: undefined,
    configuration: {}
  }
}
