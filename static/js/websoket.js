const WebSocketCpt = {
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
      var username = 'gdouyang'
      // Create a socket
      this.socket = new WebSocket('ws://' + window.location.host + '/ws/join?uname=' + username);
      // Message received on the socket
      this.socket.onmessage = (event => {
          var data = JSON.parse(event.data);
          console.log(data);
          var content = null;
          switch (data.Type) {
          case 0: // JOIN
              if (data.User == username) {
                content = 'You joined the chat room.';
              } else {
                content = data.User + ' joined the chat room.';
              }
              break;
          case 1: // LEAVE
              content = data.User + ' left the chat room.';
              break;
          case 2: // MESSAGE
              content = data.Content;
              break;
          }
          if(content) {
            this.msgList.push({content: content});
          }
      })
    },
    postConecnt(){
      this.socket.send(this.msg);
      this.msg = null;
    }
  },
  template: `
    <el-card class="box-card">
    <div slot="header" class="clearfix">
      <span>WebSocket</span>
      <el-input v-model="msg" @keyup.native.enter="postConecnt" style="width:200px;"></el-input>
      <el-button style="float: right; padding: 3px 0" type="text" @click="clear">清空消息面板</el-button>
    </div>
    <div class="text item" style="height: 500px; overflow: auto;">
      <el-alert type="info" effect="dark" style="margin-bottom: 5px;"
        v-for="(msg, index) in msgList" :key="index" :title="msg.content"></el-alert>
    </div>
    </el-card>
  ` 
}