
require(['device', 'echows', 'restful'], function(device, echows, Restful) {
  const routes = [
      { path: '/echows', component: echows },
      { path: '/restful', component: Restful },
      { path: '/device', component: device },
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
        return { visible: false }
      }
    })
});