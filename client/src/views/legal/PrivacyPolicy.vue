<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { locale } = useI18n()
const isZh = computed(() => locale.value.startsWith('zh'))
const effectiveDate = '2026-03-25'
</script>

<template>
  <div class="legal-page">
    <div class="legal-card">
      <div class="legal-header">
        <button class="back-btn" @click="router.back()">
          <i class="ri-arrow-left-line"></i>
          {{ isZh ? '返回' : 'Back' }}
        </button>
        <h1>{{ isZh ? 'RPBox 隐私政策' : 'RPBox Privacy Policy' }}</h1>
        <p class="meta">
          {{ isZh ? '生效日期' : 'Effective Date' }}: {{ effectiveDate }}
        </p>
      </div>

      <section>
        <h2>{{ isZh ? '1. 我们收集的信息' : '1. Information We Collect' }}</h2>
        <p>
          {{ isZh
            ? '我们会收集注册信息（如用户名、邮箱）、你主动提交的内容（帖子、剧情、道具、头像等），以及必要的日志与设备信息用于安全和故障排查。'
            : 'We collect registration data (such as username and email), content you submit (posts, stories, items, avatars), and necessary logs/device data for security and troubleshooting.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '2. 信息使用目的' : '2. How We Use Information' }}</h2>
        <p>
          {{ isZh
            ? '这些信息用于提供核心功能、身份验证、内容展示、通知推送、反滥用防护、性能优化与服务改进。'
            : 'This data is used to provide core features, authentication, content delivery, notifications, anti-abuse protections, performance optimization, and service improvements.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '3. 信息存储与安全' : '3. Storage and Security' }}</h2>
        <p>
          {{ isZh
            ? '我们采取合理安全措施保护数据，但任何互联网传输都无法保证绝对安全。你应妥善保管账号凭据并及时更新密码。'
            : 'We use reasonable safeguards to protect data, but no internet transmission is absolutely secure. You should keep credentials safe and update passwords promptly when needed.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '4. 信息共享' : '4. Information Sharing' }}</h2>
        <p>
          {{ isZh
            ? '除法律要求、保障平台安全或你明确授权外，我们不会向无关第三方出售你的个人信息。'
            : 'We do not sell your personal data to unrelated third parties, except where required by law, needed for platform security, or explicitly authorized by you.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '5. Cookie 与本地缓存' : '5. Cookies and Local Cache' }}</h2>
        <p>
          {{ isZh
            ? '我们可能使用本地存储与缓存（如登录态、界面偏好、图片缓存）以提升体验。你可以在设置中清理相关缓存。'
            : 'We may use local storage and cache (such as login state, UI preferences, image cache) to improve experience. You can clear cache in settings.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '6. 你的权利' : '6. Your Rights' }}</h2>
        <p>
          {{ isZh
            ? '你可以访问、更正或删除你在 RPBox 中的相关内容。若你注销或停止使用，部分日志信息可能基于合规与安全要求保留。'
            : 'You can access, correct, or delete your content in RPBox. If you close your account or stop using the service, some log data may be retained for compliance and security.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '7. 未成年人保护' : '7. Minors' }}</h2>
        <p>
          {{ isZh
            ? '如你所在地区法律要求未成年人取得监护人同意，请在注册和使用 RPBox 前完成该同意。'
            : 'If local law requires guardian consent for minors, please obtain that consent before registering for and using RPBox.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '8. 政策更新与联系' : '8. Updates and Contact' }}</h2>
        <p>
          {{ isZh
            ? '我们可能更新本政策。更新后继续使用服务即表示你已阅读并同意更新内容。若有问题，请通过项目仓库 Issue 或官方社区联系。'
            : 'We may update this policy. Continued use after updates means you have read and accepted the revised policy. For questions, contact us via project issues or official community channels.' }}
        </p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.legal-page {
  min-height: 100vh;
  padding: 24px;
  background: #eed9c4;
}

.legal-card {
  max-width: 860px;
  margin: 0 auto;
  background: #fff;
  border-radius: 16px;
  padding: 28px;
  box-shadow: 0 6px 24px rgba(75, 54, 33, 0.12);
}

.legal-header {
  margin-bottom: 20px;
}

.back-btn {
  border: 1px solid #e8dccf;
  background: #fff;
  color: #6b4a32;
  border-radius: 8px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
}

.back-btn i {
  margin-right: 4px;
}

h1 {
  margin: 14px 0 8px;
  font-size: 28px;
  color: #4b3621;
}

.meta {
  color: #8c7b70;
  font-size: 13px;
}

section {
  margin-top: 18px;
}

h2 {
  margin: 0 0 8px;
  font-size: 18px;
  color: #4b3621;
}

p {
  margin: 0;
  line-height: 1.75;
  color: #5f4a38;
  font-size: 14px;
}

@media (max-width: 720px) {
  .legal-page {
    padding: 12px;
  }

  .legal-card {
    padding: 18px;
    border-radius: 12px;
  }

  h1 {
    font-size: 22px;
  }
}
</style>
