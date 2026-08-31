<template>
  <div class="markdown">
    <h1 v-if="$route.meta.doc">{{ $route.meta.doc }}文档说明</h1>
    <MdPreview :modelValue="initialValue" :markedOptions="markedOptions" :codeFoldable="false" />
  </div>
</template>

<script>
import axios from 'axios'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
const request = axios.create({
  timeout: 20000
})
export default {
  name: 'MyMarkdown',
  components: {
    MdPreview
  },
  props: {
    type: {
      type: String,
      default: () => ''
    }
  },
  data() {
    return {
      initialValue: '',
      markedOptions: {
        baseUrl: 'static/'
      }
    }
  },
  watch: {
    '$route.fullPath': {
      handler(to, from) {
        console.log(this)
        this.getData()
      }
    }
  },
  mounted() {
    this.getData()
  },
  methods: {
    getData() {
      let t = this.type
      if (!t) {
        t = this.$route.meta.doc
      }
      request.get(`/static/doc/${t}.md`).then((resp) => {
        this.initialValue = resp.data
      })
    }
  }
}
</script>
<style lang="less" scoped>
.markdown :deep(.markdown-preview) {
  padding-top: 0px !important;
}
</style>
