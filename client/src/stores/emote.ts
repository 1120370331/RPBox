import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { listEmotePacks, resolveEmoteUrl, type EmotePack, type EmoteItem } from '@/api/emote'

export const useEmoteStore = defineStore('emote', () => {
  const packs = ref<EmotePack[]>([])
  const loaded = ref(false)
  const loading = ref(false)

  const emoteMap = computed(() => {
    const map = new Map<string, EmoteItem>()
    packs.value.forEach((pack) => {
      pack.items.forEach((item) => {
        map.set(`${pack.id}:${item.id}`, item)
      })
    })
    return map
  })

  async function loadPacks() {
    if (loading.value || loaded.value) return
    loading.value = true
    try {
      const res = await listEmotePacks()
      const next = (res.packs || []).map((pack) => ({
        ...pack,
        icon_url: resolveEmoteUrl(pack.icon_url),
        items: pack.items.map((item) => ({
          ...item,
          url: resolveEmoteUrl(item.url),
          width: item.width || 128,
          height: item.height || 128,
        })),
      }))
      packs.value = next
      loaded.value = true
    } catch (error) {
      console.error('加载表情包失败:', error)
    } finally {
      loading.value = false
    }
  }

  return {
    packs,
    loaded,
    loading,
    emoteMap,
    loadPacks,
  }
})
