# 图片加载性能优化方案

## 问题分析

当前性能瓶颈：
1. **Item 列表 API 返回完整 Base64** - `item.go:84` SELECT 包含 `preview_image`，20 条记录可能产生 10MB+ 响应
2. **无懒加载** - 所有图片一次性加载，阻塞首屏
3. **无缩略图** - 列表页加载原图（可能几 MB）
4. **头像/封面图无缓存策略** - 直接内嵌 Base64

---

## Phase 1: 前端懒加载（1-2 小时）

### 1.1 `<img>` 标签添加 loading="lazy"

**修改文件**：
- `client/src/views/community/CommunityMain.vue` - 帖子封面图、头像
- `client/src/views/market/MarketMain.vue` - 作者头像
- `client/src/views/guild/GuildDetail.vue` - 成员头像

```vue
<!-- 修改前 -->
<img :src="post.cover_image" alt="" />

<!-- 修改后 -->
<img :src="post.cover_image" alt="" loading="lazy" />
```

### 1.2 背景图懒加载组件

创建 `client/src/components/LazyBgImage.vue`：

```vue
<template>
  <div ref="container" class="lazy-bg" :class="{ loaded }" :style="bgStyle">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  src?: string
  fallback?: string
}>()

const container = ref<HTMLElement>()
const loaded = ref(false)
const shouldLoad = ref(false)

const bgStyle = computed(() => {
  if (!shouldLoad.value || !props.src) {
    return props.fallback ? { background: props.fallback } : {}
  }
  return { backgroundImage: `url(${props.src})` }
})

let observer: IntersectionObserver | null = null

onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        shouldLoad.value = true
        loaded.value = true
        observer?.disconnect()
      }
    },
    { rootMargin: '200px' }
  )
  if (container.value) {
    observer.observe(container.value)
  }
})

onUnmounted(() => observer?.disconnect())
</script>
```

**应用到**：
- `MarketMain.vue` 的 `.card-image`
- `GuildDetail.vue` 的 `.hero-bg`

---

## Phase 2: 服务端缩略图服务（1-2 天）

### 2.1 新增图片服务 API

**文件**: `server/internal/api/image.go`

**核心逻辑**：
```go
// GET /api/v1/images/:type/:id
// type: item-preview | post-cover | user-avatar | guild-banner
// 支持参数: ?w=300&h=200&q=80

func (s *Server) getImage(c *gin.Context) {
    imageType := c.Param("type")
    id := c.Param("id")
    width := c.DefaultQuery("w", "0")   // 0 表示原图
    quality := c.DefaultQuery("q", "80")

    // 1. 构造缓存路径
    cacheKey := fmt.Sprintf("%s_%s_w%s_q%s", imageType, id, width, quality)
    cachePath := filepath.Join(s.config.Storage.Path, "cache/images", cacheKey+".jpg")

    // 2. 检查缓存
    if data, err := os.ReadFile(cachePath); err == nil {
        c.Header("Cache-Control", "public, max-age=86400")
        c.Data(http.StatusOK, "image/jpeg", data)
        return
    }

    // 3. 从数据库获取原图 Base64
    base64Data := s.getOriginalImage(imageType, id)

    // 4. 解码 + 缩放 + 压缩
    img := decodeBase64(base64Data)
    if w, _ := strconv.Atoi(width); w > 0 {
        img = resize(img, w)
    }
    data := encodeJPEG(img, quality)

    // 5. 写入缓存
    os.MkdirAll(filepath.Dir(cachePath), 0755)
    os.WriteFile(cachePath, data, 0644)

    // 6. 返回
    c.Header("Cache-Control", "public, max-age=86400")
    c.Data(http.StatusOK, "image/jpeg", data)
}
```

### 2.2 修改列表 API 返回 URL 而非 Base64

**item.go - listItems()**：
```go
// 修改前：SELECT 包含 preview_image
query.Select("id, ..., preview_image, ...")

// 修改后：不返回 preview_image，改为返回 URL
query.Select("id, ..., ...")  // 移除 preview_image

// 响应中添加 URL 字段
for _, item := range items {
    item.PreviewImageURL = fmt.Sprintf("/api/v1/images/item-preview/%d?w=400", item.ID)
}
```

**类似修改**：
- `post.go` - 帖子封面图
- `guild.go` - 公会头图
- `user.go` - 用户头像（在返回用户信息时）

### 2.3 图片处理依赖

使用 Go 标准库 + 第三方库：
```go
import (
    "image"
    "image/jpeg"
    _ "image/png"
    "github.com/nfnt/resize"  // 或 "golang.org/x/image/draw"
)
```

### 2.4 前端适配

**修改 API 类型定义** (`client/src/api/`):
```typescript
interface Item {
  // preview_image?: string  // 移除
  preview_image_url?: string  // 新增
}
```

**修改组件**：
```vue
<!-- 修改前 -->
<div :style="{ backgroundImage: `url(${item.preview_image})` }">

<!-- 修改后 -->
<LazyBgImage :src="item.preview_image_url ? `${API_BASE}${item.preview_image_url}` : ''" />
```

---

## Phase 3: HTTP 缓存优化（2 小时）

### 3.1 缓存策略

| 资源类型 | 缓存时间 | 原因 |
|---------|---------|------|
| 列表缩略图 | 1 天 | 允许更新但不频繁 |
| 详情原图 | 1 小时 | 可能被编辑 |
| WoW 图标 | 1 年 | 几乎不变 |

### 3.2 服务端设置

```go
// 缩略图
c.Header("Cache-Control", "public, max-age=86400")

// 原图（详情页）
c.Header("Cache-Control", "public, max-age=3600")

// 添加 ETag 支持
etag := fmt.Sprintf(`"%x"`, md5.Sum(data))
c.Header("ETag", etag)
if c.GetHeader("If-None-Match") == etag {
    c.Status(http.StatusNotModified)
    return
}
```

### 3.3 Nginx 配置（可选，生产环境）

```nginx
location /api/v1/images/ {
    proxy_pass http://backend;
    proxy_cache images_cache;
    proxy_cache_valid 200 1d;
    proxy_cache_key $uri$is_args$args;
    add_header X-Cache-Status $upstream_cache_status;
}
```

---

## 实现顺序

1. **Phase 1** - 前端懒加载（立即见效，无后端改动）
2. **Phase 2.1** - 创建图片服务 API
3. **Phase 2.2** - 修改列表 API，返回 URL
4. **Phase 2.4** - 前端适配新 URL 格式
5. **Phase 3** - 添加缓存头和 ETag

---

## 预期效果

| 指标 | 优化前 | 优化后 |
|------|-------|-------|
| 列表 API 响应大小 | 5-10 MB | 50-100 KB |
| 首屏加载时间 | 3-5 秒 | < 1 秒 |
| 重复访问加载 | 全量重载 | 命中缓存 |
| 缩略图大小 | 原图 (500KB+) | 20-50 KB |

---

## 文件修改清单

### 新增文件
- `server/internal/api/image.go` - 图片服务 API
- `client/src/components/LazyBgImage.vue` - 懒加载背景图组件

### 修改文件
**前端**：
- `client/src/views/market/MarketMain.vue`
- `client/src/views/community/CommunityMain.vue`
- `client/src/views/guild/GuildDetail.vue`
- `client/src/api/item.ts` (类型定义)
- `client/src/api/post.ts` (类型定义)

**后端**：
- `server/internal/api/routes.go` - 添加图片路由
- `server/internal/api/item.go` - 列表不返回 Base64
- `server/internal/api/post.go` - 列表不返回 Base64
- `server/internal/api/guild.go` - 列表不返回 Base64
- `server/internal/model/model.go` - 添加 URL 字段
- `server/go.mod` - 添加 resize 依赖
