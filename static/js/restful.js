define(function() {
  'use strict';
  return {
    data: function () {
      return {
        msgList: []
      }
    },
    mounted(){
      this.init();
    },
    methods: {
      clear() {
        this.msgList = [];
      },
      init(){
        // Create a socket
        this.socket = new WebSocket('ws://' + window.location.host + '/ws/echo');
        // Message received on the socket
        this.socket.onmessage = (event => {
            var content = event.data;
            if(content) {
              this.msgList.push({content: content});
            }
        })
        this.socket.onopen = (event => {
          this.msgList.push({content: 'echo websocket connected '+ new Date()})
        })
        this.socket.onclose = (event => {
          this.msgList.push({content: 'echo websocket close '+ new Date()})
        })
      },
    },
    template: `
      <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span>Restful</span>
        <el-button style="float: right; padding: 3px 0" type="text" @click="clear">清空消息面板</el-button>
      </div>
      <div class="text item">
        <el-alert type="info" effect="dark" style="margin-bottom: 5px;"
          v-for="(msg, index) in msgList" :key="index" :title="msg.content"></el-alert>
      </div>
      </el-card>
    ` 
  }
});