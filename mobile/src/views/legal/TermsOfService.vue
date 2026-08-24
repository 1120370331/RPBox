<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { locale } = useI18n()
const isZh = computed(() => locale.value.startsWith('zh'))
const effectiveDate = '2026-08-24'
const supportUrl = 'https://github.com/1120370331/RPBox/issues'

const sections = computed(() => isZh.value
  ? [
      {
        title: '1. 服务与条款接受',
        paragraphs: [
          'RPBox 为魔兽世界 RP 玩家提供人物卡管理与同步、剧情归档、社区分享、RP 数据库和道具展示等功能。注册或使用服务即表示你同意遵守本条款；如果你无法依法同意，请勿注册或使用需要账号的功能。',
        ],
      },
      {
        title: '2. 账号与安全',
        paragraphs: [
          '你应提供准确的注册信息，妥善保管账号凭据，并对账号下的活动负责。请勿共享凭据、冒用他人身份、未经授权访问账号，或绕过安全和审核机制。发现账号被盗用时，请立即通过支持渠道反馈。',
        ],
      },
      {
        title: '3. 用户内容与禁止行为',
        paragraphs: [
          '你对自己提交的内容负责。不得发布违法、侵权、欺诈、垃圾广告、恶意代码、骚扰、仇恨、威胁、冒充他人、暴露他人隐私、剥削或危害未成年人，以及其他明显令人反感或破坏社区安全的内容。',
        ],
      },
      {
        title: '4. 举报、屏蔽与社区安全',
        paragraphs: [
          '在帖子、评论、道具、RP 数据库作品和剧情等详情页，你可以使用「举报」提交原因和说明，并可在可用位置屏蔽作者；人物卡详情支持举报或屏蔽持有人，公会详情支持举报并可屏蔽会长。「个人中心 → 屏蔽用户」可管理屏蔽名单。无法使用应用内举报时，可通过下方支持链接反馈。',
          '举报可能由版主或管理员审核。我们可根据内容和风险采取隐藏或删除内容、限制功能、禁言、暂停或终止账号等措施，并会尽力及时处理有效的安全举报。',
        ],
      },
      {
        title: '5. 内容权利与服务许可',
        paragraphs: [
          '你保留对原创内容拥有的权利，并确认你有权提交其中包含的文字、图片和其他材料。为提供 RPBox 服务，你授予我们在服务运行所需范围内存储、转换格式、传输、展示和审核该内容的非独占许可。该许可不转移你的所有权。',
        ],
      },
      {
        title: '6. 内容和账号删除',
        paragraphs: [
          '你可以删除自己控制的内容，并通过「个人中心 → 删除账号」提交账号删除。已审核通过、已发布且公开的 RP 数据库知识作品可能在账号匿名化并清除媒体归属后保留，具体删除与保存规则见《隐私政策》。对于违反本条款、存在安全风险或依法必须处理的内容和账号，我们也可以采取删除或限制措施。',
        ],
      },
      {
        title: '7. 赞助展示与数字内容',
        paragraphs: [
          '当前手机应用中的赞助者名单仅用于致谢，不在应用内出售数字内容、功能或权益，也不会因展示在名单中而解锁应用功能。若未来提供付费数字内容，将遵守适用的应用商店支付规则并另行说明。',
        ],
      },
      {
        title: '8. 服务变更与可用性',
        paragraphs: [
          '我们可能为维护、安全、法律要求或产品调整而变更、暂停或停止部分服务。我们会努力保持服务可用，但无法保证服务始终不中断或完全无错误。请为重要的本地数据保留合理备份。',
        ],
      },
      {
        title: '9. 独立项目与商标',
        paragraphs: [
          'RPBox 是独立的社区工具，与 Blizzard Entertainment 无隶属、赞助或认可关系。World of Warcraft、魔兽世界及相关名称和标识归其各自权利人所有。你不得以 RPBox 或第三方商标误导他人。',
        ],
      },
      {
        title: '10. 条款变更',
        paragraphs: [
          '我们可能更新本条款，并在本页标注新的生效日期；适用法律要求时会提供额外通知。若你不同意更新后的条款，应停止使用相关服务，并可删除自己的内容或账号。',
        ],
      },
    ]
  : [
      {
        title: '1. Service and Acceptance',
        paragraphs: [
          'RPBox provides character-card management and synchronization, story archives, community sharing, an RP Database, and item showcases for World of Warcraft role-players. By registering or using the service, you agree to follow these terms. If you cannot legally agree, do not register or use account-based features.',
        ],
      },
      {
        title: '2. Account and Security',
        paragraphs: [
          'Provide accurate registration information, protect your credentials, and take responsibility for activity under your account. Do not share credentials, impersonate others, access accounts without authorization, or bypass security or moderation controls. Report suspected account compromise promptly through the support channel.',
        ],
      },
      {
        title: '3. User Content and Prohibited Conduct',
        paragraphs: [
          'You are responsible for content you submit. Do not post illegal or infringing material, fraud, spam, malicious code, harassment, hate, threats, impersonation, another person\'s private information, content that exploits or endangers minors, or other clearly objectionable material that undermines community safety.',
        ],
      },
      {
        title: '4. Reporting, Blocking, and Community Safety',
        paragraphs: [
          'On detail screens for posts, comments, items, RP Database works, and stories, you can use Report to submit a reason and explanation and, where available, block an author. Character-card details support reporting or blocking the owner, and guild details support reporting and blocking the guild owner. You can manage blocks through Profile → Blocked Users. If in-app reporting is unavailable, use the support link below.',
          'Reports may be reviewed by moderators or administrators. Depending on the content and risk, we may hide or remove content, restrict features, mute, suspend, or terminate an account. We aim to address actionable safety reports promptly.',
        ],
      },
      {
        title: '5. Content Rights and Service License',
        paragraphs: [
          'You retain the rights you hold in original content and confirm that you have the right to submit included text, images, and other material. To provide RPBox, you grant us a non-exclusive license to store, format, transmit, display, and moderate that content only as needed to operate the service. This license does not transfer ownership.',
        ],
      },
      {
        title: '6. Content and Account Deletion',
        paragraphs: [
          'You can delete content you control and submit account deletion through Profile → Delete Account. An approved, published, and public RP Database knowledge work may remain after account anonymization and removal of media attribution; see the Privacy Policy for the exact deletion and retention rules. We may also remove or restrict content or accounts that violate these terms, create a security risk, or must be addressed by law.',
        ],
      },
      {
        title: '7. Sponsor Display and Digital Content',
        paragraphs: [
          'The current mobile app\'s sponsor list is an acknowledgement only. It does not sell or unlock digital content, features, or benefits in the app. If paid digital content is offered in the future, we will follow applicable app-store payment rules and provide additional terms.',
        ],
      },
      {
        title: '8. Service Changes and Availability',
        paragraphs: [
          'We may change, suspend, or discontinue parts of the service for maintenance, security, legal requirements, or product changes. We work to keep the service available but cannot guarantee uninterrupted or error-free operation. Keep reasonable backups of important local data.',
        ],
      },
      {
        title: '9. Independent Project and Trademarks',
        paragraphs: [
          'RPBox is an independent community tool and is not affiliated with, sponsored by, or endorsed by Blizzard Entertainment. World of Warcraft and related names and marks belong to their respective owners. Do not use RPBox or third-party marks to mislead others.',
        ],
      },
      {
        title: '10. Changes to These Terms',
        paragraphs: [
          'We may update these terms and will show a new effective date on this page. We will provide additional notice where required by applicable law. If you disagree with updated terms, stop using the relevant service and, if desired, delete your content or account.',
        ],
      },
    ])
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ isZh ? '服务条款' : 'Terms of Service' }}</h1>
    </header>
    <div class="sub-body legal-body">
      <div class="legal-card">
        <p class="meta">{{ isZh ? '生效日期' : 'Effective Date' }}: {{ effectiveDate }}</p>

        <section v-for="section in sections" :key="section.title">
          <h2>{{ section.title }}</h2>
          <p v-for="paragraph in section.paragraphs" :key="paragraph">{{ paragraph }}</p>
        </section>

        <section>
          <h2>{{ isZh ? '11. 联系与申诉' : '11. Contact and Appeals' }}</h2>
          <p>
            {{ isZh ? '条款问题、内容安全反馈或对处理结果的申诉，请通过 ' : 'For terms questions, safety reports, or an appeal of a moderation action, use ' }}
            <a :href="supportUrl" target="_blank" rel="noopener noreferrer">GitHub Issues</a>{{ isZh ? ' 联系我们。该渠道公开可见，请勿提交密码、访问令牌或其他敏感信息。' : '. This channel is publicly visible, so do not post passwords, access tokens, or other sensitive information.' }}
          </p>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.legal-body {
  padding-bottom: calc(20px + var(--safe-bottom, 0px));
}

.legal-card {
  background: var(--color-card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  border: 1px solid rgba(75, 54, 33, 0.08);
  padding: 16px 14px;
}

.meta {
  margin: 0 0 12px;
  color: var(--color-text-muted);
  font-size: 12px;
}

section + section {
  margin-top: 14px;
}

h2 {
  margin: 0 0 6px;
  font-size: 14px;
  color: var(--text-dark);
}

p {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
}

p + p {
  margin-top: 8px;
}

a {
  color: var(--color-primary);
  text-underline-offset: 2px;
}
</style>
