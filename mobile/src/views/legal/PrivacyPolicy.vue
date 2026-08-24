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
        title: '1. 适用范围与我们处理的信息',
        paragraphs: [
          '本政策说明 RPBox 手机应用及相关服务如何处理信息。我们遵循必要性原则，仅处理提供你所请求功能所需的信息。',
        ],
        items: [
          '账号信息：用户名、邮箱和内部账号 ID。',
          '你主动提交的内容：帖子、剧情、道具、RP 数据库作品、人物卡、评论、头像和图片。你选择公开发布的内容会按产品设计向其他用户展示。',
          '服务端安全与运行日志：认证和限流日志可能包含 IP 地址、用户名或用户 ID、请求路径、请求时间、响应状态以及登录失败、限流等安全事件，用于安全防护、反滥用和故障排查。',
          '设备本地数据：登录状态、语言和界面偏好、图片缓存保存在设备上；除非某项功能明确传输相关信息，否则这些本地数据不会作为服务端资料收集。',
        ],
      },
      {
        title: '2. 相机、照片与可选通知',
        paragraphs: [
          '只有在你主动拍摄或选择媒体时，RPBox 才会调用相机或照片选择功能；iOS 仍可能根据系统组件行为显示照片库授权提示。无论提示范围如何，只有你最终选中的媒体才会上传。撤销系统权限不会自动删除此前已上传的内容，你可以在对应内容页另行删除。',
          '当前版本不包含推送通知注册能力，也不收集推送令牌。如果未来版本启用通知，只有在你授权后，我们才会处理推送令牌和通知偏好，并仅用于发送 RPBox 通知；届时本政策和相应商店披露也会同步更新。',
        ],
      },
      {
        title: '3. 信息使用目的',
        paragraphs: [
          '我们使用上述信息进行账号注册与身份验证、人物卡和剧情同步、社区内容发布与展示、内容审核与反滥用、安全防护、用户支持和故障排查；通知功能启用时，还会用于发送你允许的服务通知。',
        ],
      },
      {
        title: '4. 公开展示、共享与跟踪',
        paragraphs: [
          '你公开发布的内容以及与内容一起展示的用户名、头像会被其他用户看到。你未公开的邮箱不会因社区展示而公开。',
          '为运行存储、传输和内容交付，基础设施或服务提供方可能仅在提供相应服务所必需的范围内处理信息；我们也可能在法律要求或保护用户、平台安全所必需时披露有限信息。',
          '我们不出售个人信息。当前版本未集成广告 SDK，也不使用你的信息在其他公司的应用或网站间跟踪你。',
        ],
      },
      {
        title: '5. 保存期限与删除',
        paragraphs: [
          '你可在「个人中心」删除自己发布的内容，并通过「个人中心 → 删除账号」发起服务端删除。账号删除会移除帖子、道具、剧情、人物卡、私有人物资料、备份以及非公开的 RP 数据库数据。',
          '为保留已进入公共知识库的贡献，同时满足「已审核通过、已发布、公开」条件的 RP 数据库知识作品可能继续保留；此时用户账号壳会匿名化，保留作品的媒体归属信息会被清除。',
          '除此之外，只有法律要求或保护服务安全所必需的少量安全、审计日志可能在必要期限内保留，且不会用于其他目的。你还可以在「关于 RPBox → 图片缓存」清除本地图片缓存。',
        ],
      },
      {
        title: '6. 你的选择与权利',
        paragraphs: [
          '你可以在应用内访问、更正和删除自己控制的资料与内容。你可以通过系统设置撤销相机、照片或通知权限，以停止相应的可选处理；如需处理无法在应用内完成的访问、更正或删除请求，请使用下方公开支持渠道。',
        ],
      },
      {
        title: '7. 本地存储与安全',
        paragraphs: [
          'RPBox 使用设备本地存储保存登录状态、偏好和图片缓存；这些数据只有在功能明确需要传输时才会离开设备。我们采取与风险相称的技术和管理措施保护数据，但任何网络传输或存储方式都无法保证绝对安全。请勿分享账号凭据。',
        ],
      },
      {
        title: '8. 未成年人',
        paragraphs: [
          '如果你所在地区要求未成年人取得监护人同意，请在注册或提交个人信息前取得该同意。若你认为未成年人未经适当同意提交了信息，请通过支持渠道联系我们。',
        ],
      },
      {
        title: '9. 政策变更',
        paragraphs: [
          '我们会在本页更新生效日期，并在适用法律要求时提供额外通知或重新取得同意。若你不同意变更，可以停止相关可选处理、删除内容或删除账号。',
        ],
      },
    ]
  : [
      {
        title: '1. Scope and Information We Process',
        paragraphs: [
          'This policy explains how the RPBox mobile app and related services process information. We follow data-minimization principles and process only what is needed to provide the features you request.',
        ],
        items: [
          'Account data: username, email address, and internal account ID.',
          'Content you choose to submit: posts, stories, items, RP Database works, character cards, comments, avatars, and images. Content you choose to publish is shown to other users as part of the service.',
          'Server security and operational logs: authentication and rate-limit logs may include IP address, username or user ID, request path, request time, response status, and security events such as failed sign-in attempts or rate limiting. We use these logs for security, anti-abuse, and troubleshooting.',
          'On-device data: login state, language and interface preferences, and image cache are stored on the device. Unless a feature explicitly transmits related information, this local data is not collected as server-side profile data.',
        ],
      },
      {
        title: '2. Camera, Photos, and Optional Notifications',
        paragraphs: [
          'RPBox invokes camera or photo selection only after you choose to take or select media. iOS may still display a photo-library authorization prompt based on system-component behavior. Regardless of the prompt scope, only media you ultimately select is uploaded. Revoking a system permission does not remove content already uploaded; you can delete that content from its relevant screen.',
          'The current release has no push-notification registration capability and does not collect a push token. If a future release enables notifications, we will process a push token and notification preferences only after permission, only to deliver RPBox notifications, and will update this policy and the relevant store disclosures.',
        ],
      },
      {
        title: '3. How We Use Information',
        paragraphs: [
          'We use the information above for account registration and authentication, character-card and story synchronization, publishing and delivering community content, moderation and anti-abuse, security, user support, and troubleshooting. If notifications are enabled, we also use it to send service notifications you allow.',
        ],
      },
      {
        title: '4. Public Visibility, Sharing, and Tracking',
        paragraphs: [
          'Content you publish, together with the username and avatar displayed with that content, is visible to other users. Your non-public email address is not made public through community display.',
          'Infrastructure or service providers may process information only as needed to operate storage, transmission, and content delivery. We may also disclose limited information where required by law or necessary to protect users or the service.',
          'We do not sell personal information. The current release contains no advertising SDK, and we do not use your information to track you across other companies\' apps or websites.',
        ],
      },
      {
        title: '5. Retention and Deletion',
        paragraphs: [
          'You can delete content you control in the app and start server-side deletion through Profile → Delete Account. Account deletion removes posts, items, stories, character cards, private profiles, backups, and non-public RP Database data.',
          'To preserve contributions already accepted into the public knowledge base, an RP Database knowledge work may remain only when it is approved, published, and public. In that case, the user account shell is anonymized and media attribution on the retained work is cleared.',
          'Apart from that exception, a narrow set of security or audit logs may be retained only when and for as long as legally required or necessary to protect the service, and not for unrelated purposes. You can clear the local image cache through About RPBox → Image Cache.',
        ],
      },
      {
        title: '6. Your Choices and Rights',
        paragraphs: [
          'You can access, correct, and delete profile information and content you control in the app. You can revoke camera, photo, or notification permission in system settings to stop the related optional processing. For an access, correction, or deletion request that cannot be completed in the app, use the public support channel below.',
        ],
      },
      {
        title: '7. Local Storage and Security',
        paragraphs: [
          'RPBox uses on-device storage for login state, preferences, and image cache; this data leaves the device only when a feature explicitly requires transmission. We apply safeguards proportionate to the risk, but no network transmission or storage method can be guaranteed completely secure. Do not share your account credentials.',
        ],
      },
      {
        title: '8. Minors',
        paragraphs: [
          'Where local law requires guardian consent for a minor, obtain that consent before registration or submission of personal information. If you believe a minor submitted information without appropriate consent, contact us through the support channel.',
        ],
      },
      {
        title: '9. Changes to This Policy',
        paragraphs: [
          'We will update the effective date on this page and provide additional notice or obtain renewed consent where required by applicable law. If you disagree with a change, you may stop the related optional processing, delete content, or delete your account.',
        ],
      },
    ])
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <button class="back-btn" @click="router.back()"><i class="ri-arrow-left-line" /></button>
      <h1>{{ isZh ? '隐私政策' : 'Privacy Policy' }}</h1>
    </header>
    <div class="sub-body legal-body">
      <div class="legal-card">
        <p class="meta">{{ isZh ? '生效日期' : 'Effective Date' }}: {{ effectiveDate }}</p>

        <section v-for="section in sections" :key="section.title">
          <h2>{{ section.title }}</h2>
          <p v-for="paragraph in section.paragraphs" :key="paragraph">{{ paragraph }}</p>
          <ul v-if="section.items">
            <li v-for="item in section.items" :key="item">{{ item }}</li>
          </ul>
        </section>

        <section>
          <h2>{{ isZh ? '10. 联系我们' : '10. Contact Us' }}</h2>
          <p>
            {{ isZh ? '隐私问题、权利请求或安全问题请通过 ' : 'For privacy questions, rights requests, or security concerns, use ' }}
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

p + p,
ul {
  margin-top: 8px;
}

ul {
  padding-left: 20px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

li + li {
  margin-top: 4px;
}

a {
  color: var(--color-primary);
  text-underline-offset: 2px;
}
</style>
