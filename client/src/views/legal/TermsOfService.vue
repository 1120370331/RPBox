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
        <h1>{{ isZh ? 'RPBox 服务条款' : 'RPBox Terms of Service' }}</h1>
        <p class="meta">
          {{ isZh ? '生效日期' : 'Effective Date' }}: {{ effectiveDate }}
        </p>
      </div>

      <section>
        <h2>{{ isZh ? '1. 服务说明' : '1. Service Scope' }}</h2>
        <p>
          {{ isZh
            ? 'RPBox 为魔兽世界 RP 玩家提供人物资料管理、剧情记录、社区互动与相关扩展服务。你使用 RPBox 即表示同意按本条款使用服务。'
            : 'RPBox provides profile management, story archives, community features, and related services for WoW RP players. By using RPBox, you agree to these terms.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '2. 账号与安全' : '2. Account and Security' }}</h2>
        <p>
          {{ isZh
            ? '你需要对账号行为和凭据安全负责。请勿共享账号或以任何方式规避平台安全机制。'
            : 'You are responsible for account activities and credential security. Do not share accounts or attempt to bypass platform protections.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '3. 用户内容与行为规范' : '3. User Content and Conduct' }}</h2>
        <p>
          {{ isZh
            ? '你发布的内容应合法、真实且不侵犯他人权益，不得包含违法、骚扰、仇恨、诈骗、恶意脚本或破坏社区秩序的内容。'
            : 'Your content must be lawful and must not infringe rights of others. Illegal, abusive, hateful, fraudulent, malicious-script, or disruptive content is prohibited.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '4. 知识产权' : '4. Intellectual Property' }}</h2>
        <p>
          {{ isZh
            ? '你对自己上传和创作的内容保留权利，并授权 RPBox 在提供服务范围内进行存储、展示与传输。RPBox 品牌与平台代码受相关法律保护。'
            : 'You retain rights to your own uploaded/created content and grant RPBox a limited license to store, display, and transmit it for service operation. RPBox branding and platform code are legally protected.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '5. 服务可用性与免责声明' : '5. Availability and Disclaimer' }}</h2>
        <p>
          {{ isZh
            ? '我们会尽力保障服务稳定，但不承诺零中断、零故障。对于不可抗力、第三方服务故障或用户操作导致的数据问题，RPBox 在法律允许范围内承担有限责任。'
            : 'We strive for stable service but do not guarantee uninterrupted or error-free operation. RPBox has limited liability to the extent permitted by law for force majeure, third-party failures, or user-caused data issues.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '6. 违规处理' : '6. Enforcement' }}</h2>
        <p>
          {{ isZh
            ? '若发现违规，我们可采取内容下架、限制功能、禁言、封禁账号等措施，并保留追究责任的权利。'
            : 'For violations, we may remove content, limit features, mute, suspend, or ban accounts, and reserve the right to pursue further action.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '7. 条款更新' : '7. Updates to These Terms' }}</h2>
        <p>
          {{ isZh
            ? '我们可能更新本条款。更新后继续使用服务，视为你同意更新内容。若你不同意，应停止使用相关服务。'
            : 'We may update these terms. Continued use after updates means acceptance of the revised terms. If you disagree, stop using the related services.' }}
        </p>
      </section>

      <section>
        <h2>{{ isZh ? '8. 联系我们' : '8. Contact' }}</h2>
        <p>
          {{ isZh
            ? '如对本条款有疑问，可通过项目仓库 Issue 或官方社区渠道联系我们。'
            : 'If you have questions about these terms, contact us through project issues or official community channels.' }}
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
