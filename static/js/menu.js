// contents of main.js:
require.config({
    paths: {
        device: 'device',
        restful: 'restful',
        echows: 'echows',
        material: 'material',
    }
});

require(['device', 'echows', 'restful', 'material'], 
function(device, echows, Restful, material) {
  const routes = [
      { path: '/echows', component: echows },
      { path: '/restful', component: Restful },
      { path: '/device', component: device },
      { path: '/material', component: material },
    ]
    
    // 3. 创建 router 实例，然后传 `routes` 配置
    // 你还可以传别的配置参数, 不过先这么简单着吧。
    const router = new VueRouter({
        routes // (缩写) 相当于 routes: routes
    })
    new Vue({
      el: '#app',
      router: router,
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
        <i class="el-icon-connection"></i>
        <span slot="title">Push Echo</span>
      </el-menu-item>
      <el-menu-item index="restful">
        <i class="el-icon-monitor"></i>
        <span slot="title">Restful</span>
      </el-menu-item>
      <el-submenu index="2">
        <template slot="title">
          <i class="el-icon-mobile-phone"></i>
          <span slot="title">设备中心</span>
        </template>
        <el-menu-item-group>
        <template slot="title">设备</template>
          <el-menu-item index="device">LED</el-menu-item>
        </el-menu-item-group>
      </el-submenu>
      <el-menu-item index="material">
        <i class="el-icon-files"></i>
        <span slot="title">素材</span>
      </el-menu-item>
    </el-menu>
	`
});
