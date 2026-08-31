<template>
  <div>
    <el-card class="location-card">
      <baidu-map
        class="map"
        :zoom="zoom"
        :scroll-wheel-zoom="true"
        :mapClick="false"
        @moveend="search"
        @ready="handler"
        ak="g5IoseVn5ZlLawf4cL5lTyIfZb9y6GOQ"
      >
        <bm-geolocation
          anchor="BMAP_ANCHOR_BOTTOM_RIGHT"
          :showAddressBar="true"
          :autoLocation="true"
        />
      </baidu-map>
      <el-drawer
        title="设备"
        placement="right"
        width="800"
        :visible="GetDetailStatus"
        @close="back"
      >
        <DeviceDetail ref="DeviceDetail" v-if="GetDetailStatus" @back="back" />
      </el-drawer>
    </el-card>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
// import BaiduMap from 'vue-baidu-map/components/map/Map.vue'
// import BmGeolocation from 'vue-baidu-map/components/controls/Geolocation.vue'
import DeviceDetail from './DetailIndex.vue'

const defautSearchObj = {
  name: ''
}
export default {
  components: {
    // BaiduMap,
    // BmGeolocation,
    DeviceDetail
  },
  data() {
    return {
      searchObj: _.cloneDeep(defautSearchObj),
      zoom: 15,
      GetDetailStatus: false
    }
  },
  created() {},
  methods: {
    handler({ BMap, map }) {
      console.log(BMap, map)
      this.map = map
      const sysConfig = this.$store.getters.sysConfig
      if (sysConfig && sysConfig.defaultLocation) {
        const l = sysConfig.defaultLocation
        // map.setCenter({ lng: l.longitude, lat: l.latitude })
        map.setCenter(new window.BMap.Point(l.longitude, l.latitude))
      } else {
        var myCity = new BMap.LocalCity()
        myCity.get((result) => {
          var cityName = result.name
          map.setCenter(cityName)
        })
      }
      this.zoom = 15
      this.search()
    },
    search() {
      if (!this.map) {
        return
      }
      const bounds = this.map.getBounds()
      const northEast = bounds.getNorthEast()
      const southWest = bounds.getSouthWest()
      const s = {
        startLongitude: southWest.lng,
        endLongitude: northEast.lng,
        startLatitude: southWest.lat,
        endLatitude: northEast.lat
      }
      if (!this.markerList) {
        this.markerList = []
      }
      this.$http.post('device-extend/map', s).then((resp) => {
        if (resp.success) {
          const map = this.map
          _.forEach(resp.result, (d) => {
            var point = new window.BMap.Point(d.longitude, d.latitude)
            var marker = new window.BMap.Marker(point)
            marker.deviceId = d.id
            const m = _.find(this.markerList, (m) => m.deviceId === d.id)
            if (m) {
              map.removeOverlay(m)
              this.markerList = _.filter(this.markerList, (m) => m.deviceId !== d.id)
            }
            this.markerList.push(marker)
            // event{type, target}
            marker.addEventListener('click', (event) => {
              const id = event.target.deviceId
              console.log(id + '被点击')
              this.$router.push({ name: this.$route.name, query: { id: id } })
              this.GetDetailStatus = true
            })
            map.addOverlay(marker)
          })
        }
      })
    },
    back() {
      this.$router.push({ name: this.$route.name, query: {} })
      this.GetDetailStatus = false
    }
  }
}
</script>

<style lang="less" scoped>
.location-card {
}
.map {
  width: 100%;
  min-height: 500px;
  height: ~'calc(100vh - 100px)';
}
</style>
