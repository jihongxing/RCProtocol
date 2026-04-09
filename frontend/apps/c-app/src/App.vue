<script setup lang="ts">
import { onLaunch, onShow } from '@dcloudio/uni-app'
import { useAuth } from '@rcprotocol/state'

const { isLoggedIn, loadFromStorage } = useAuth()

onLaunch(() => {
  loadFromStorage()
})

onShow(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  if (currentPage) {
    const route = (currentPage as any).route || ''
    if (route.startsWith('pages/vault') && !isLoggedIn.value) {
      uni.reLaunch({ url: '/pages/login' })
    }
  }
})
</script>

<style>
page {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 28rpx;
  color: #333;
  background-color: #f5f5f5;
}
</style>
