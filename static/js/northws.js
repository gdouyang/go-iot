const NorthWebSocketCpt = {
  data: function () {
    return {
      socket: null,
      msg: null,
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
      this.socket = new WebSocket('ws://' + window.location.host + '/ws/north?evt=online-status&evt=switch-status');
      // Message received on the socket
      this.socket.onmessage = (event => {
          var data = JSON.parse(event.data);
          console.log(data);
          var content = null;
          switch (data.Type) {
          case 2: // MESSAGE
              content = data.Content;
              break;
          }
          if(content) {
            this.msgList.push({content: content});
          }
      })
      this.socket.onopen = (event => {
        this.msgList.push({content: '北向websocket connected '+ new Date()})
      })
      this.socket.onclose = (event => {
        this.msgList.push({content: '北向websocket close '+ new Date()})
      })
    },
  },
  template: `
    <el-card class="box-card">
    <div slot="header" class="clearfix">
      <span>North WebSocket</span>
      <el-button style="float: right; padding: 3px 0" type="text" @click="clear">清空消息面板</el-button>
    </div>
    <div class="text item" style="height: 500px; overflow: auto;">
      <el-alert type="info" effect="dark" style="margin-bottom: 5px;"
        v-for="(msg, index) in msgList" :key="index" :title="msg.content"></el-alert>
    </div>
    </el-card>
  ` 
}