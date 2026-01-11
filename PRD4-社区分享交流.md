# PRD4: 社区分享交流平台

## 1. 概述

### 1.1 背景
当前RP玩家的剧情分享主要依赖：
- 百度贴吧金色平原吧
- NGA论坛

存在的问题：
- 没有按公会/人物/剧情线归档
- 缺乏便捷的检索机制
- 好的剧情难以被发现
- 内容分散，难以形成社区氛围

### 1.2 目标
构建RP玩家专属社区平台，提供：
- 结构化的内容归档（公会/人物/剧情线）
- 强大的搜索和推荐
- 互动功能（点赞、评论、收藏）
- 公会主页和成员管理

## 2. 用户故事

| 编号 | 故事 | 优先级 |
|------|------|--------|
| US-01 | 我想发布我的RP剧情战报 | P0 |
| US-02 | 我想搜索特定公会/角色的剧情 | P0 |
| US-03 | 我想关注感兴趣的创作者 | P1 |
| US-04 | 我想创建公会主页展示公会故事 | P1 |

## 3. 功能需求

### 3.1 内容发布 (P0)
- 支持 Markdown 格式
- 图片上传
- 标签系统（公会、角色、剧情线）
- 草稿保存

### 3.2 内容检索 (P0)
- 全文搜索
- 按标签筛选
- 按时间/热度排序
- 高级搜索（多条件组合）

### 3.3 社交互动 (P1)
- 点赞、收藏、分享
- 评论系统
- 关注创作者
- 消息通知

### 3.4 公会系统 (P1)
- 公会主页创建
- 成员管理
- 公会剧情时间线
- 公会公告

## 4. 数据结构

```typescript
interface Post {
  id: string;
  authorId: number;
  title: string;
  content: string;
  tags: string[];
  guildId?: string;
  likes: number;
  views: number;
  status: 'draft' | 'published';
  createdAt: Date;
}

interface Guild {
  id: string;
  name: string;
  description: string;
  ownerId: number;
  members: number[];
  createdAt: Date;
}
```

## 5. 里程碑

| 阶段 | 功能 |
|------|------|
| M1 | 内容发布 + 基础展示 |
| M2 | 搜索 + 标签系统 |
| M3 | 社交互动 |
| M4 | 公会系统 |
