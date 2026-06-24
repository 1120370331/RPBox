# RP数据库后端逻辑与数据模型设计

本文基于 `PRD3-1-RP数据库.md`、`style-demos/rp_db.html` 以及当前 Go 后端实现整理。目标是先把 RP 数据库的领域边界、数据模型、审核与 API 逻辑设计清楚，后续可按阶段实现。

## 1. 结论

RP 数据库不应复用现有 `Item` 主表。

现有 `Item` 是 TRP3 Extended 自制道具市场，核心字段是 `import_code`、`raw_data`、下载、评分和一键导入。RP 数据库的对象是魔兽世界真实物品、玩具、饰品、装备、幻化方案等，核心价值是展示 RP 效果、获取攻略、外部资料链接、合集和个人收集清单。

建议新增 `rpdb_` 前缀的一组领域表：

- `rpdb_entries`: RP 数据库主条目，表示一个真实游戏对象或 RP 展示资源。
- `rpdb_external_refs`: Wowhead 等外部数据库链接。
- `rpdb_media`: 截图、GIF、视频链接、嵌入内容。
- `rpdb_guides` / `rpdb_guide_steps`: 获取攻略与可导出的路径步骤。
- `rpdb_sets` / `rpdb_set_entries`: 公开主题合集。
- `rpdb_lists` / `rpdb_list_entries`: 用户个人收集清单。
- `rpdb_favorites` / `rpdb_views`: 用户互动。
- `rpdb_revisions`: 创建和编辑审核流。
- 复用现有 `tags`，新增 `category = "rpdb"`，通过 `rpdb_entry_tags` 关联。

家园地图可以共享同一套媒体、审核、举报和搜索思路，但建议独立为 `home_entries`，不要塞进 RP 数据库主表。

## 2. 当前后端可复用能力

当前后端是 Gin + GORM，模型集中在 `server/internal/model`，路由集中在 `server/internal/api/routes.go`，迁移由 `server/internal/database/database.go` 的 `AutoMigrate` 加少量手写 SQL 完成。

可复用能力：

- 图片上传和存储：`normalizeAndStoreImageValue`、`saveUploadedImage`、`/uploads/*filepath`。
- 缩略图和缓存：`/api/v1/images/:type/:id`，支持 `w`、`q`、`v` 和 ETag。
- 审核体系：帖子、道具、公会都已有 `status`、`review_status`、`reviewer_id`、`review_comment`、`reviewed_at`。
- 版主中心：`/api/v1/moderator/review/*`、`logAdminAction`、审核通知。
- 举报和隐藏：`ContentReport`、`UserHiddenContent`、`safety.go` 的 `reportTarget*`。
- 用户展示：用户名颜色、等级、头像 URL 可以复用。

需要扩展能力：

- 图片类型新增 `rpdb-entry-preview`，或让 `rpdb_media` 直接走 `/uploads`。
- 举报目标新增 `rpdb_entry`、`rpdb_guide`，可选 `rpdb_media`。
- 标签预设新增 `category = "rpdb"`。
- 账号删除时，RP 数据库的已审核公共条目不应直接删除，建议匿名化作者；草稿和待审核内容可以删除。

## 3. 领域边界

### RP 数据库条目

一个 `rpdb_entry` 表示一个可被检索、收藏、加入清单的 RP 展示资源。它通常映射到魔兽真实对象：

- 可使用物品
- 饰品
- 玩具
- 装备
- 幻化方案
- 未来可扩展到坐骑、宠物、法术效果、地点或活动奖励

条目主表只保存高频筛选、展示和排序字段。变化快、结构不稳定或一对多的信息放附表或 JSONB。

### 道具市场

现有 `items` 继续只负责 TRP3 Extended 自制内容。两者可以在前端导航和搜索结果中并列展示，但数据库不要合表。

### 家园地图

家园地图是住宅系统分享社区，建议独立模型：

- `home_entries`
- `home_media`
- `home_favorites`

它可以复用媒体、审核、举报、搜索基础设施，但不要与 `rpdb_entries` 混用 `category`。

## 4. 数据模型

下面是推荐模型。实现时可放在 `server/internal/model/rpdb.go`，同属 `model` 包，并显式定义 `TableName()`，避免 GORM 把 `RPDBEntry` 命名成不易读的表名。

### 4.1 RPDBEntry

主条目保存可检索、可筛选、可排序的稳定字段。

```go
type RPDBEntry struct {
    ID          uint   `gorm:"primarykey" json:"id"`
    CanonicalKey string `gorm:"size:128;uniqueIndex;not null" json:"canonical_key"`

    ExternalType string `gorm:"size:32;index" json:"external_type"` // wow_item|toy|equipment|transmog|spell|manual
    ExternalID   string `gorm:"size:64;index" json:"external_id"`

    Name        string `gorm:"size:256;not null;index" json:"name"`
    Slug        string `gorm:"size:256;uniqueIndex" json:"slug"`
    Category    string `gorm:"size:32;index" json:"category"`    // usable|trinket|toy|equipment|transmog|other
    Subcategory string `gorm:"size:64;index" json:"subcategory"`
    Quality     string `gorm:"size:32;index" json:"quality"`     // common|rare|epic|legendary|unknown
    Icon        string `gorm:"size:128" json:"icon"`

    PreviewImage          string     `gorm:"type:text" json:"preview_image"`
    PreviewImageUpdatedAt *time.Time `json:"preview_image_updated_at,omitempty"`

    Summary           string `gorm:"size:512" json:"summary"`
    EffectDescription string `gorm:"type:text" json:"effect_description"`
    LoreNote          string `gorm:"type:text" json:"lore_note"`
    AcquisitionSummary string `gorm:"type:text" json:"acquisition_summary"`

    AvailabilityStatus string `gorm:"size:32;index" json:"availability_status"` // available|seasonal|limited|removed|unknown
    UnavailableReason  string `gorm:"size:256" json:"unavailable_reason"`
    BindType           string `gorm:"size:32;index" json:"bind_type"` // none|pickup|equip|account|quest|unknown
    IsAccountLimited   bool   `gorm:"default:false;index" json:"is_account_limited"`
    IsUnique           bool   `gorm:"default:false" json:"is_unique"`
    Faction            string `gorm:"size:32;index" json:"faction"` // alliance|horde|neutral|unknown
    Expansion          string `gorm:"size:64;index" json:"expansion"`
    Patch              string `gorm:"size:32" json:"patch"`
    MinLevel           int    `gorm:"default:0" json:"min_level"`

    RestrictionsJSON string `gorm:"type:jsonb" json:"restrictions_json"` // class/race/profession 等
    ExtraJSON        string `gorm:"type:jsonb" json:"extra_json"`        // 后续扩展字段

    AuthorID       *uint  `gorm:"index" json:"author_id"` // 可空，账号注销后保留公共知识
    AuthorSnapshot string `gorm:"size:80" json:"author_snapshot"`
    MaintainerID   *uint  `gorm:"index" json:"maintainer_id"`

    Status       string `gorm:"size:20;default:draft;index" json:"status"` // draft|pending|published|archived|removed
    IsPublic     bool   `gorm:"default:true;index" json:"is_public"`
    ReviewStatus string `gorm:"size:20;default:pending;index" json:"review_status"` // none|pending|approved|rejected
    ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
    ReviewComment string     `gorm:"size:512" json:"review_comment"`
    ReviewedAt    *time.Time `json:"reviewed_at"`

    Version        int `gorm:"default:1" json:"version"`
    ViewCount      int `gorm:"default:0" json:"view_count"`
    FavoriteCount  int `gorm:"default:0" json:"favorite_count"`
    ChecklistCount int `gorm:"default:0" json:"checklist_count"`
    GuideCount     int `gorm:"default:0" json:"guide_count"`
    MediaCount     int `gorm:"default:0" json:"media_count"`

    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

设计要点：

- `CanonicalKey` 是去重核心，例如 `wow:item:19019`、`wow:toy:12345`、`rpbox:manual:<uuid>`。
- `ExternalType + ExternalID` 便于接 Wowhead、暴雪 API、WarcraftDB 等来源。
- `RestrictionsJSON` 和 `ExtraJSON` 用 JSONB 承接不稳定字段，避免主表频繁加列。
- `AuthorID` 可空，服务账号注销匿名化。
- 公共查询必须过滤 `status = published AND review_status = approved AND is_public = true`。

### 4.2 RPDBExternalRef

外部资料链接不要只放一个 `wowhead_url` 字段，后续可能有多个来源和语言。

```go
type RPDBExternalRef struct {
    ID         uint      `gorm:"primarykey" json:"id"`
    EntryID    uint      `gorm:"uniqueIndex:idx_rpdb_external_ref;not null" json:"entry_id"`
    Source     string    `gorm:"size:32;uniqueIndex:idx_rpdb_external_ref;not null" json:"source"` // wowhead|warcraftdb|bnet|other
    ExternalID string    `gorm:"size:64" json:"external_id"`
    Locale     string    `gorm:"size:16" json:"locale"`
    URL        string    `gorm:"type:text;not null" json:"url"`
    IsPrimary  bool      `gorm:"default:false;index" json:"is_primary"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

### 4.3 RPDBMedia

媒体是 RP 数据库体验核心，应独立于主表。截图、GIF、视频链接和嵌入内容统一建模。

```go
type RPDBMedia struct {
    ID        uint   `gorm:"primarykey" json:"id"`
    EntryID   uint   `gorm:"index;not null" json:"entry_id"`
    AuthorID  *uint  `gorm:"index" json:"author_id"`

    MediaType string `gorm:"size:20;index" json:"media_type"` // image|gif|video|embed
    URL       string `gorm:"type:text;not null" json:"url"`   // /uploads 或外部链接
    ThumbnailURL string `gorm:"type:text" json:"thumbnail_url"`
    Caption   string `gorm:"size:256" json:"caption"`
    SortOrder int    `gorm:"default:0" json:"sort_order"`
    Width     int    `gorm:"default:0" json:"width"`
    Height    int    `gorm:"default:0" json:"height"`
    DurationSeconds int `gorm:"default:0" json:"duration_seconds"`
    MetaJSON  string `gorm:"type:jsonb" json:"meta_json"`

    ReviewStatus string `gorm:"size:20;default:pending;index" json:"review_status"`
    ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
    ReviewComment string     `gorm:"size:512" json:"review_comment"`
    ReviewedAt    *time.Time `json:"reviewed_at"`

    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

首图逻辑：

- 主表 `PreviewImage` 可保存上传 URL 或留空。
- 若主表没有预览图，详情页取第一张已审核 `rpdb_media` 作为封面。
- 若使用通用图片服务，新增 image type `rpdb-entry-preview`。

### 4.4 RPDBGuide / RPDBGuideStep

获取攻略建议支持多个用户贡献，但详情页可有一个主攻略。

```go
type RPDBGuide struct {
    ID        uint   `gorm:"primarykey" json:"id"`
    EntryID   uint   `gorm:"index;not null" json:"entry_id"`
    AuthorID  *uint  `gorm:"index" json:"author_id"`
    Title     string `gorm:"size:128;not null" json:"title"`
    Content   string `gorm:"type:text" json:"content"`
    ContentType string `gorm:"size:20;default:markdown" json:"content_type"`
    IsPrimary bool   `gorm:"default:false;index" json:"is_primary"`

    UsefulCount int `gorm:"default:0" json:"useful_count"`
    NotUsefulCount int `gorm:"default:0" json:"not_useful_count"`

    Status       string `gorm:"size:20;default:pending;index" json:"status"` // pending|published|archived
    ReviewStatus string `gorm:"size:20;default:pending;index" json:"review_status"`
    ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
    ReviewComment string     `gorm:"size:512" json:"review_comment"`
    ReviewedAt    *time.Time `json:"reviewed_at"`

    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type RPDBGuideStep struct {
    ID        uint   `gorm:"primarykey" json:"id"`
    GuideID   uint   `gorm:"index;not null" json:"guide_id"`
    SortOrder int    `gorm:"default:0" json:"sort_order"`
    Title     string `gorm:"size:128" json:"title"`
    Body      string `gorm:"type:text" json:"body"`
    Zone      string `gorm:"size:128;index" json:"zone"`
    MapID     string `gorm:"size:64" json:"map_id"`
    CoordinatesJSON string `gorm:"type:jsonb" json:"coordinates_json"` // [{x,y,label}]
    TomTomCommand   string `gorm:"type:text" json:"tomtom_command"`
    MetaJSON        string `gorm:"type:jsonb" json:"meta_json"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

`TomTomCommand` 可以存缓存，也可以由后端根据 `CoordinatesJSON` 动态生成。建议第一版动态生成，避免坐标规则变更后批量迁移。

### 4.5 RPDBSet / RPDBSetEntry

公开合集用于“相关道具”“主题收集”“一套幻化方案”等场景，不等同用户个人清单。

```go
type RPDBSet struct {
    ID          uint   `gorm:"primarykey" json:"id"`
    AuthorID    *uint  `gorm:"index" json:"author_id"`
    Name        string `gorm:"size:128;not null" json:"name"`
    Description string `gorm:"type:text" json:"description"`
    CoverImage  string `gorm:"type:text" json:"cover_image"`
    SetType     string `gorm:"size:32;index" json:"set_type"` // theme|transmog|event|route|other
    ItemCount   int    `gorm:"default:0" json:"item_count"`
    IsPublic    bool   `gorm:"default:true;index" json:"is_public"`
    Status      string `gorm:"size:20;default:pending;index" json:"status"`
    ReviewStatus string `gorm:"size:20;default:pending;index" json:"review_status"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type RPDBSetEntry struct {
    ID      uint `gorm:"primarykey" json:"id"`
    SetID   uint `gorm:"uniqueIndex:idx_rpdb_set_entry;not null" json:"set_id"`
    EntryID uint `gorm:"uniqueIndex:idx_rpdb_set_entry;not null" json:"entry_id"`
    Role    string `gorm:"size:32" json:"role"` // required|optional|variant
    SortOrder int `gorm:"default:0" json:"sort_order"`
    Note string `gorm:"size:256" json:"note"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 4.6 RPDBList / RPDBListEntry

用户个人收集任务清单。默认清单可以在第一次“加入清单”时自动创建。

```go
type RPDBList struct {
    ID          uint   `gorm:"primarykey" json:"id"`
    UserID      uint   `gorm:"index;not null" json:"user_id"`
    Name        string `gorm:"size:128;not null" json:"name"`
    Description string `gorm:"type:text" json:"description"`
    IsDefault   bool   `gorm:"default:false;index" json:"is_default"`
    IsPublic    bool   `gorm:"default:false;index" json:"is_public"`
    ItemCount   int    `gorm:"default:0" json:"item_count"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type RPDBListEntry struct {
    ID      uint `gorm:"primarykey" json:"id"`
    ListID  uint `gorm:"uniqueIndex:idx_rpdb_list_entry;not null" json:"list_id"`
    EntryID uint `gorm:"uniqueIndex:idx_rpdb_list_entry;not null" json:"entry_id"`
    Status  string `gorm:"size:20;default:wanted;index" json:"status"` // wanted|farming|owned|ignored
    Priority int   `gorm:"default:0" json:"priority"`
    Quantity int   `gorm:"default:1" json:"quantity"`
    Note     string `gorm:"size:512" json:"note"`
    SortOrder int  `gorm:"default:0" json:"sort_order"`
    AcquiredAt *time.Time `json:"acquired_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

清单导出：

- `format=json`: 完整结构，给 RPBox 或未来插件导入。
- `format=csv`: 玩家手动整理。
- `format=tomtom`: 从主攻略步骤坐标生成 TomTom 宏命令，缺坐标的条目列入 `missing_coordinates`。

### 4.7 RPDBFavorite / RPDBView

独立表保留用户行为，主表冗余计数字段用于列表性能。

```go
type RPDBFavorite struct {
    ID uint `gorm:"primarykey" json:"id"`
    EntryID uint `gorm:"uniqueIndex:idx_rpdb_favorite_user;not null" json:"entry_id"`
    UserID uint `gorm:"uniqueIndex:idx_rpdb_favorite_user;not null" json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}

type RPDBView struct {
    ID uint `gorm:"primarykey" json:"id"`
    EntryID uint `gorm:"uniqueIndex:idx_rpdb_view_user;not null" json:"entry_id"`
    UserID uint `gorm:"uniqueIndex:idx_rpdb_view_user;not null" json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 4.8 RPDBRevision

已发布条目的编辑不应直接覆盖，普通用户提交修订，版主审核后应用。这样可以支持多人协作和回滚。

```go
type RPDBRevision struct {
    ID uint `gorm:"primarykey" json:"id"`
    EntryID *uint `gorm:"index" json:"entry_id"` // 创建新条目时为空
    ProposerID uint `gorm:"index;not null" json:"proposer_id"`
    Action string `gorm:"size:20;index;not null" json:"action"` // create|update|archive|merge
    BaseVersion int `gorm:"default:0" json:"base_version"`
    PayloadJSON string `gorm:"type:jsonb;not null" json:"payload_json"`
    ChangeSummary string `gorm:"size:512" json:"change_summary"`
    Status string `gorm:"size:20;default:pending;index" json:"status"` // pending|approved|rejected|applied
    ReviewerID *uint `gorm:"index" json:"reviewer_id"`
    ReviewComment string `gorm:"size:512" json:"review_comment"`
    ReviewedAt *time.Time `json:"reviewed_at"`
    AppliedAt *time.Time `json:"applied_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

审核通过时在事务中：

1. 校验 `base_version` 是否仍等于主表版本。
2. 将 `PayloadJSON` 允许的字段应用到 `rpdb_entries` 和附表。
3. `version += 1`。
4. 写入管理日志。
5. 更新搜索索引任务。

## 5. 索引和迁移建议

`AutoMigrate` 能创建基础表，但列表、筛选和唯一约束建议手写索引。

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_rpdb_entries_external_unique
ON rpdb_entries(external_type, external_id)
WHERE COALESCE(BTRIM(external_type), '') <> ''
  AND COALESCE(BTRIM(external_id), '') <> '';

CREATE INDEX IF NOT EXISTS idx_rpdb_entries_public_list
ON rpdb_entries(status, review_status, is_public, category, updated_at);

CREATE INDEX IF NOT EXISTS idx_rpdb_entries_flags
ON rpdb_entries(category, quality, availability_status, bind_type);

CREATE INDEX IF NOT EXISTS idx_rpdb_media_entry_review
ON rpdb_media(entry_id, review_status, sort_order);

CREATE INDEX IF NOT EXISTS idx_rpdb_guides_entry_primary
ON rpdb_guides(entry_id, is_primary, review_status);

CREATE INDEX IF NOT EXISTS idx_rpdb_list_entries_user_entry
ON rpdb_list_entries(entry_id);

CREATE INDEX IF NOT EXISTS idx_rpdb_entries_extra_json
ON rpdb_entries USING GIN ((extra_json::jsonb));
```

搜索可以分两阶段：

- 第一阶段：数据库 `ILIKE` 搜索 `name`、`summary`、`effect_description`、`acquisition_summary`。
- 第二阶段：接 MeiliSearch，写入 `rpdb_entry` 搜索文档，字段包括名称、分类、效果、攻略摘要、标签、稀有度、绝版状态、外部 ID。

如果要在 PostgreSQL 内增强搜索，可启用 `pg_trgm`：

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_rpdb_entries_name_trgm
ON rpdb_entries USING GIN (name gin_trgm_ops);
```

是否启用 `pg_trgm` 取决于生产数据库权限；没有权限时不要阻塞启动。

## 6. API 设计

### 6.1 公开/登录均可访问

公开列表只返回审核通过内容。若登录用户访问，可附带 `is_favorited`、`in_checklist`。

```http
GET /api/v1/rpdb/entries
```

查询参数：

- `search`
- `category`
- `quality`
- `availability_status`
- `bind_type`
- `is_account_limited`
- `is_unavailable`
- `tag_id`
- `set_id`
- `external_type`
- `sort`: `updated_at|popular|favorite|created_at`
- `page`
- `page_size`

详情：

```http
GET /api/v1/rpdb/entries/:id
GET /api/v1/rpdb/entries/:id/media
GET /api/v1/rpdb/entries/:id/guides
GET /api/v1/rpdb/entries/:id/sets
GET /api/v1/rpdb/sets
GET /api/v1/rpdb/sets/:id
```

### 6.2 需要登录

```http
POST   /api/v1/rpdb/entries
PATCH  /api/v1/rpdb/entries/:id
DELETE /api/v1/rpdb/entries/:id

POST   /api/v1/rpdb/entries/:id/media
DELETE /api/v1/rpdb/entries/:id/media/:mediaId

POST   /api/v1/rpdb/entries/:id/guides
PATCH  /api/v1/rpdb/guides/:guideId
POST   /api/v1/rpdb/guides/:guideId/feedback

POST   /api/v1/rpdb/entries/:id/favorite
DELETE /api/v1/rpdb/entries/:id/favorite

GET    /api/v1/rpdb/lists
POST   /api/v1/rpdb/lists
PATCH  /api/v1/rpdb/lists/:listId
DELETE /api/v1/rpdb/lists/:listId
POST   /api/v1/rpdb/lists/:listId/entries/:entryId
PATCH  /api/v1/rpdb/lists/:listId/entries/:entryId
DELETE /api/v1/rpdb/lists/:listId/entries/:entryId
GET    /api/v1/rpdb/lists/:listId/export?format=tomtom
```

创建逻辑：

- 草稿：`status = draft`，`review_status = none`。
- 普通用户发布：`status = pending`，`review_status = pending`。
- 版主/管理员发布：`status = published`，`review_status = approved`。
- 如果 `external_type + external_id` 已存在，返回冲突，并提示用户提交修订。

编辑逻辑：

- 作者编辑自己的草稿：直接更新。
- 普通用户编辑已发布条目：创建 `rpdb_revisions`。
- 版主编辑已发布条目：可直接更新，但仍写 `rpdb_revisions` 或 `AdminActionLog` 作为审计。

### 6.3 版主接口

```http
GET  /api/v1/moderator/review/rpdb-entries
POST /api/v1/moderator/review/rpdb-entries/:id

GET  /api/v1/moderator/review/rpdb-revisions
POST /api/v1/moderator/review/rpdb-revisions/:id

GET  /api/v1/moderator/review/rpdb-media
POST /api/v1/moderator/review/rpdb-media/:id

GET  /api/v1/moderator/manage/rpdb-entries
POST /api/v1/moderator/manage/rpdb-entries/:id/hide
DELETE /api/v1/moderator/manage/rpdb-entries/:id
```

审核通过后：

- 更新状态。
- 写 `AdminActionLog`，`target_type = "rpdb_entry"`。
- 给贡献者发送系统通知。
- 更新搜索索引。

## 7. 核心业务流程

### 7.1 创建条目

1. 接收基础字段、外部链接、媒体、标签、攻略摘要。
2. 校验名称、分类、URL、媒体类型和字段长度。
3. 生成 `canonical_key`：
   - 有外部 ID：`wow:<external_type>:<external_id>`。
   - 无外部 ID：`rpbox:manual:<uuid>`。
4. 查重：
   - `canonical_key` 冲突直接返回已存在条目。
   - 名称近似只返回 warning，不阻止提交。
5. 图片统一转 `/uploads` 或保留外部 URL。
6. 普通用户进入审核，版主自动通过。

### 7.2 编辑条目

1. 草稿直接更新。
2. 已发布内容创建 `rpdb_revisions`，不直接覆盖。
3. 审核通过时校验 `base_version`。
4. 冲突时要求贡献者基于最新版本重新提交。

### 7.3 加入清单

1. 用户没有默认清单时创建“我的 RP 收集清单”。
2. `rpdb_list_entries` 使用 `(list_id, entry_id)` 唯一约束避免重复。
3. 更新 `rpdb_entries.checklist_count`。
4. 支持状态：想要、正在刷、已获得、忽略。

### 7.4 导出 TomTom

1. 读取清单条目。
2. 查每个条目的主攻略。
3. 读取攻略步骤坐标。
4. 生成 TomTom 命令。
5. 缺坐标条目返回在 `missing_coordinates`，前端提示用户手工查看攻略。

返回示例：

```json
{
  "format": "tomtom",
  "content": "/way 84 42.1 65.3 古老的琥珀封印\n/way 84 51.4 71.0 奖励箱子",
  "missing_coordinates": [
    { "entry_id": 12, "name": "幻象：虚空之拥" }
  ]
}
```

### 7.5 删除和账号注销

RP 数据库是公共知识库，不建议用户删除账号时硬删除已审核公共条目。

建议规则：

- 草稿、待审核修订、待审核媒体：删除。
- 已发布条目、已发布攻略：保留内容，将 `author_id = null`，`author_snapshot = "已注销用户"`。
- 用户个人清单、收藏、浏览记录：删除。
- 如果用户明确要求删除其贡献内容，可进入人工处理或批量改为 `archived`。

## 8. 与现有系统的接入点

### 图片服务

新增：

- `image.go` 支持 `rpdb-entry-preview`。
- `image_url_helpers.go` 增加 `rpdbEntryPreviewURL`。
- 图片变更时更新 `preview_image_updated_at`。

媒体列表里上传的图片/GIF可以直接返回 `/uploads` URL；只有封面预览需要走缩略图服务。

### 标签

复用 `Tag`：

- 新增预设标签 `Category = "rpdb"`。
- 新增 `RPDBEntryTag`。
- `getPresetTags` 已支持按 `category` 过滤，可直接用 `/api/v1/tags/preset?category=rpdb`。

### 举报和隐藏

在 `safety.go` 增加：

- `reportTargetRPDBEntry = "rpdb_entry"`
- `reportTargetRPDBGuide = "rpdb_guide"`

扩展：

- `resolveReportTarget`
- `validateReportReviewAction`
- `deleteReportedTarget`
- `listContentReports` 预览构建

`UserHiddenContent.TargetType` 目前注释只列 post/item/comment，但字段长度足够，新增 target type 即可。

### 管理日志和通知

`AdminActionLog.TargetType` 是 `size:20`，`rpdb_entry`、`rpdb_guide` 足够。

`Notification.TargetType` 是 `size:20`，也足够。

建议通知类型继续使用 `system`，内容中说明 RP 数据库条目审核结果，减少通知类型膨胀。

### 活跃度

第一版不必接积分，避免刷条目。若接入：

- 首次通过审核的条目给少量经验。
- 攻略被标记有用达到阈值后给贡献奖励。
- 清单、收藏、浏览不加分。

## 9. 家园地图后端建议

家园地图不和 RP 数据库共主表，但可复用同样的审核/媒体模式。

核心表：

- `home_entries`: 标题、描述、服务器/区域、战网昵称、住宅分享代码、作者、状态、审核字段、浏览/收藏计数。
- `home_media`: 截图、视频、排序、审核字段。
- `home_favorites`: 收藏。

住宅分享代码目前是预期功能，应设计为可空字段：

```go
ShareCode string `gorm:"size:512" json:"share_code"`
ShareCodeStatus string `gorm:"size:20;default:unavailable" json:"share_code_status"` // unavailable|provided|verified
```

不要把 `home_entries` 做成 `rpdb_entries.category = home`，否则搜索、筛选、攻略、外部数据库字段都会变成大量空值。

## 10. 分阶段实现建议

### 阶段一：RPDB MVP

- 新增模型和迁移。
- 列表、详情、创建、草稿、发布审核。
- 外部链接。
- 封面图。
- 收藏和默认清单。
- 基础筛选和搜索。

### 阶段二：媒体和攻略

- 多媒体上传和外链。
- 攻略与步骤。
- TomTom 导出。
- 有用/无用反馈。

### 阶段三：协作与治理

- `rpdb_revisions` 修订审核。
- 举报、隐藏、版主管理。
- 搜索索引。
- 账号注销处理。

### 阶段四：家园地图

- 独立 `home_entries`。
- 住宅媒体展示。
- 战网昵称和未来分享代码。

## 11. 实现时的注意事项

- 列表接口不要返回大字段：`effect_description`、`lore_note`、`extra_json`、攻略正文、媒体列表只在详情取。
- 所有写接口用事务，特别是主表计数、清单、收藏、审核。
- 所有计数字段都要能重算，避免删除/审核回滚后漂移。
- URL 字段必须限制只允许 `http://`、`https://`、`/uploads/`、`uploads/`，避免奇怪协议。
- JSON 文件和 JSONB 字符串仍使用标准 ASCII 引号，避免项目文档中提到的曲引号问题。
- 第一版不要引入外键约束，保持与当前模型风格一致；但要用唯一索引和查询索引保证数据质量。
- 新模型建议拆到 `model/rpdb.go`，不要继续膨胀 `model.go`。
