// contents of main.js:
require.config({
    paths: {
      led: 'led',
        restful: 'restful',
        echows: 'echows',
        material: 'material',
        agent: 'agent',
        mediaTransport: 'mediaTransport',
        mediaServer: 'mediaServer',
        instance: 'instance',
        camera: 'camera',
        plan: 'plan',
    }
});

require(['led', 'echows', 'restful', 'material', 'agent','mediaTransport','mediaServer','instance','camera', 'plan'], 
function(led, echows, Restful, material, agent,mediaTransport,mediaServer,instance,camera,plan) {
  const routes = [
      { path: '/echows', component: echows },
      { path: '/restful', component: Restful },
      { path: '/led', component: led },
      { path: '/material', component: material },
      { path: '/agent', component: agent },
      { path: '/mediaTransport', component: mediaTransport },
      { path: '/mediaServer', component: mediaServer },
      { path: '/instance', component: instance },
      { path: '/camera', component: camera },
      { path: '/plan', component: plan },
    ]
    
    // 3. 创建 router 实例，然后传 `routes` 配置
    // 你还可以传别的配置参数, 不过先这么简单着吧。
    const router = new VueRouter({
        routes // (缩写) 相当于 routes: routes
    })
    new Vue({
      el: '#app',
      router: router,
      mounted () {
        window.onresize = ()=>{
          this.elMainStyle.height = document.documentElement.clientHeight+'px'
        }
      },
      data: function() {
        return {
          visible: false,
          elMainStyle:{
            height: document.documentElement.clientHeight+'px'
          }
        }
      }
    })
});

Vue.component('my-menu', {
	method:{
		toPage(path){
			this.$router.push({path: path})
		}
	},
	template: `
		<el-menu default-active="2" background-color="#545c64" text-color="#fff" :router="true" style="height: 100%;">
      <el-menu-item index="echows">
        <i class="el-icon-chat-line-square"></i>
        <span slot="title">Push Echo</span>
      </el-menu-item>
      <el-menu-item index="restful">
        <i class="el-icon-chat-line-round"></i>
        <span slot="title">Restful</span>
      </el-menu-item>
      <el-menu-item index="agent">
        <i class="el-icon-connection"></i>
        <span slot="title">Agent</span>
      </el-menu-item>
      <el-submenu index="2">
        <template slot="title">
          <i class="el-icon-mobile-phone"></i>
          <span slot="title">设备中心</span>
        </template>
        <el-menu-item-group>
        <template slot="title">设备</template>
          <el-menu-item index="led">LED</el-menu-item>
        </el-menu-item-group>
      </el-submenu>
      <el-menu-item index="material">
        <i class="el-icon-files"></i>
        <span slot="title">素材</span>
      </el-menu-item>
      <el-menu-item index="plan">
        <i class="el-icon-files"></i>
        <span slot="title">计划</span>
      </el-menu-item>
      <el-submenu index="3">
        <template slot="title">
          <i class="el-icon-mobile-phone"></i>
          <span slot="title">监控中心</span>
        </template>
        <el-menu-item-group>
          <el-menu-item index="mediaTransport">推流实例</el-menu-item>
          <el-menu-item index="mediaServer">流媒体服务</el-menu-item>
          <el-menu-item index="instance">链路管理</el-menu-item>
          <el-menu-item index="camera">监控设备</el-menu-item>
        </el-menu-item-group>
      </el-submenu>
    </el-menu>
	`
});
