
var _logger = logger;

simulator.bindHandler("/invoke-function", function (message, session) {
    var messageId = message.messageId;
    var functionId = message.function;

    if (functionId === 'mockChildConnect') {
        var deviceId = message.args[0];
        session.sendMessage("/child-device-connect", JSON.stringify({
            messageId: new Date().getTime(),
            childDeviceId: deviceId,
            timestamp: new Date().getTime(),
            success: true
        }))
    }

    session.sendMessage("/invoke-function-reply", JSON.stringify({
        messageId: messageId,
        output: "success",
        timestamp: new Date().getTime(),
        success: true
    }))
});

simulator.bindHandler("/read-property", function (message, session) {
    var messageId = message.messageId;

    session.sendMessage("/read-property-reply", JSON.stringify({
        messageId: messageId,
        timestamp: new Date().getTime(),
        properties: {"name": "1234"},
        success: true
    }))
});

//子设备操作
simulator.bindChildHandler("/read-property", function (message, session) {
    var messageId = message.messageId;

    session.sendChilDeviceMessage("/read-property-reply", message.deviceId, JSON.stringify({
        messageId: messageId,
        timestamp: new Date().getTime(),
        properties: {"name": "3456"},
        success: true
    }))
});


simulator.onEvent(function (index, session) {
    session.sendMessage("/report-property",JSON.stringify({
//	"properties": {
        temperature: Number(((Math.random() * 32) + 1).toFixed(2)),
        light: Number(((Math.random() * 100) + 1).toFixed(0)),
        voltage: 220,
        current: Number(((Math.random() * 100) + 1).toFixed(0)),
        humidity: Number(((Math.random() * 100) + 1).toFixed(2))
//	}
    }))
});

simulator.onConnect(function (session) {
   // _logger.info("[{}]:连接成功",session.auth.clientId)
});

simulator.onAuth(function(index,auth){
//   auth.setClientId("simulator-device-"+index);
   auth.setUsername("admin");
   auth.setPassword("admin");

});
