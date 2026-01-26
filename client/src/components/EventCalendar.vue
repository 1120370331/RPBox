<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listEvents, type EventItem } from '@/api/post'

const router = useRouter()
const { t } = useI18n()
const events = ref<EventItem[]>([])
const loading = ref(false)
const currentDate = ref(new Date())

// 当前年月
const currentYear = computed(() => currentDate.value.getFullYear())
const currentMonth = computed(() => currentDate.value.getMonth())

// 月份名称 - 使用 i18n
const monthNames = computed(() => [
  t('common.calendar.months.january'),
  t('common.calendar.months.february'),
  t('common.calendar.months.march'),
  t('common.calendar.months.april'),
  t('common.calendar.months.may'),
  t('common.calendar.months.june'),
  t('common.calendar.months.july'),
  t('common.calendar.months.august'),
  t('common.calendar.months.september'),
  t('common.calendar.months.october'),
  t('common.calendar.months.november'),
  t('common.calendar.months.december'),
])

// 星期名称 - 使用 i18n
const weekDays = computed(() => [
  t('common.calendar.weekdays.sun'),
  t('common.calendar.weekdays.mon'),
  t('common.calendar.weekdays.tue'),
  t('common.calendar.weekdays.wed'),
  t('common.calendar.weekdays.thu'),
  t('common.calendar.weekdays.fri'),
  t('common.calendar.weekdays.sat'),
])

// 获取当月第一天是星期几
const firstDayOfMonth = computed(() => {
  return new Date(currentYear.value, currentMonth.value, 1).getDay()
})

// 获取当月天数
const daysInMonth = computed(() => {
  return new Date(currentYear.value, currentMonth.value + 1, 0).getDate()
})

// 生成日历格子
const calendarDays = computed(() => {
  const days: { date: number; isCurrentMonth: boolean; events: EventItem[] }[] = []

  // 上个月的天数
  const prevMonthDays = new Date(currentYear.value, currentMonth.value, 0).getDate()

  // 填充上个月的日期
  for (let i = firstDayOfMonth.value - 1; i >= 0; i--) {
    days.push({ date: prevMonthDays - i, isCurrentMonth: false, events: [] })
  }

  // 填充当月日期
  for (let i = 1; i <= daysInMonth.value; i++) {
    const dateStr = `${currentYear.value}-${String(currentMonth.value + 1).padStart(2, '0')}-${String(i).padStart(2, '0')}`
    const dayEvents = events.value.filter(e => {
      if (!e.event_start_time) return false
      return e.event_start_time.startsWith(dateStr)
    })
    days.push({ date: i, isCurrentMonth: true, events: dayEvents })
  }

  // 填充下个月的日期（补齐6行）
  const remaining = 42 - days.length
  for (let i = 1; i <= remaining; i++) {
    days.push({ date: i, isCurrentMonth: false, events: [] })
  }

  return days
})

// 今天的日期
const today = new Date()
const isToday = (date: number) => {
  return currentYear.value === today.getFullYear() &&
         currentMonth.value === today.getMonth() &&
         date === today.getDate()
}

// 切换月份
function prevMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value - 1, 1)
  loadEvents()
}

function nextMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value + 1, 1)
  loadEvents()
}

function goToToday() {
  currentDate.value = new Date()
  loadEvents()
}

// 加载活动
async function loadEvents() {
  loading.value = true
  try {
    const start = `${currentYear.value}-${String(currentMonth.value + 1).padStart(2, '0')}-01`
    const lastDay = new Date(currentYear.value, currentMonth.value + 1, 0).getDate()
    const end = `${currentYear.value}-${String(currentMonth.value + 1).padStart(2, '0')}-${lastDay}`

    const res = await listEvents(start, end)
    events.value = res.events || []
  } catch (error) {
    console.error('加载活动失败:', error)
  } finally {
    loading.value = false
  }
}

// 查看活动详情
function viewEvent(event: EventItem) {
  router.push({ name: 'post-detail', params: { id: event.id } })
}

onMounted(() => {
  loadEvents()
})
</script>

<template>
  <div class="event-calendar">
    <!-- Loading Overlay -->
    <div v-if="loading" class="loading-overlay">
      <div class="loader"></div>
      <span class="loading-text">{{ $t('common.calendar.syncingEvents') }}</span>
    </div>

    <!-- Header Section -->
    <div class="calendar-header">
      <!-- Month Title & Year -->
      <div class="header-left">
        <h2 class="calendar-label">{{ $t('common.calendar.title') }}</h2>
        <div class="month-year">
          <h1 class="current-month">{{ monthNames[currentMonth] }}</h1>
          <span class="current-year">{{ currentYear }}</span>
        </div>
      </div>

      <!-- Controls -->
      <div class="header-controls">
        <button class="nav-btn" @click="prevMonth">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
          </svg>
        </button>
        <button class="today-btn" @click="goToToday">{{ $t('common.calendar.today') }}</button>
        <button class="nav-btn" @click="nextMonth">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
          </svg>
        </button>
      </div>
    </div>

    <!-- Calendar Body -->
    <div class="calendar-body">
      <!-- Weekday Headers -->
      <div class="weekday-header">
        <div v-for="day in weekDays" :key="day" class="weekday">{{ day }}</div>
      </div>

      <!-- Days Grid -->
      <div class="calendar-grid">
        <div
          v-for="(day, index) in calendarDays"
          :key="index"
          class="calendar-day"
          :class="{
            'other-month': !day.isCurrentMonth,
            'is-today': day.isCurrentMonth && isToday(day.date)
          }"
          :style="{ animationDelay: `${index * 10}ms` }"
        >
          <span
            class="day-number"
            :class="{ 'today-badge': day.isCurrentMonth && isToday(day.date) }"
          >{{ day.date }}</span>

          <div v-if="day.events.length > 0" class="day-events">
            <a
              v-for="event in day.events.slice(0, 2)"
              :key="event.id"
              class="event-item"
              :class="event.event_type"
              :title="event.title"
              @click="viewEvent(event)"
            >
              <span class="event-title">{{ event.title }}</span>
            </a>
            <div v-if="day.events.length > 2" class="more-events">
              {{ $t('common.calendar.moreEvents', { n: day.events.length - 2 }) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ========== Main Container ========== */
.event-calendar {
  position: relative;
  width: 100%;
  background: var(--color-panel-bg, #FFFFFF);
  box-shadow: var(--shadow-lg, 0 12px 40px rgba(75, 54, 33, 0.12));
  border-radius: 4px 48px 4px 48px;
  overflow: hidden;
  border: 1px solid var(--color-border-light, rgba(255, 255, 255, 0.6));
}

/* ========== Loading Overlay ========== */
.loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 50;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(4px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.loader {
  width: 24px;
  height: 24px;
  border: 3px solid var(--color-primary-light, rgba(128, 64, 48, 0.1));
  border-left-color: var(--color-secondary, #804030);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-secondary, #804030);
  letter-spacing: 0.05em;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ========== Header Section ========== */
.calendar-header {
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  border-bottom: 1px solid var(--color-border, #E8DCCF);
  background: linear-gradient(to right, var(--color-panel-bg, #FFFFFF), var(--color-card-bg, #FAF6F3));
}

@media (min-width: 768px) {
  .calendar-header {
    flex-direction: row;
    justify-content: space-between;
    align-items: flex-end;
  }
}

.header-left {
  display: flex;
  flex-direction: column;
}

.calendar-label {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-accent, #D4A373);
  letter-spacing: 0.1em;
  text-transform: uppercase;
  margin: 0 0 4px 0;
}

.month-year {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.current-month {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--color-text-main, #2C1810);
  line-height: 1;
  margin: 0;
  font-family: 'Playfair Display', Georgia, serif;
}

@media (min-width: 768px) {
  .current-month {
    font-size: 3rem;
  }
}

.current-year {
  font-size: 1.5rem;
  font-weight: 300;
  color: var(--color-text-secondary, #8C7B70);
  font-family: 'Playfair Display', Georgia, serif;
}

/* ========== Controls ========== */
.header-controls {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--color-panel-bg, #FFFFFF);
  padding: 6px;
  border-radius: 9999px;
  border: 1px solid var(--color-border, #E8DCCF);
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(75, 54, 33, 0.05));
}

.nav-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: transparent;
  border-radius: 9999px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-main, #2C1810);
  transition: all 0.2s;
}

.nav-btn:hover {
  background: var(--color-background, #EED9C4);
  color: var(--color-secondary, #804030);
}

.nav-btn:focus {
  outline: none;
  box-shadow: 0 0 0 2px var(--color-primary-light, rgba(128, 64, 48, 0.2));
}

.nav-icon {
  width: 20px;
  height: 20px;
}

.today-btn {
  padding: 8px 16px;
  border: none;
  border-top: none;
  border-bottom: none;
  border-left: 1px solid var(--color-border-light, rgba(232, 220, 207, 0.5));
  border-right: 1px solid var(--color-border-light, rgba(232, 220, 207, 0.5));
  background: transparent;
  color: var(--color-text-main, #2C1810);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.2s;
  margin: 0 4px;
}

.today-btn:hover {
  color: var(--color-secondary, #804030);
}

/* ========== Calendar Body ========== */
.calendar-body {
  background: var(--color-panel-bg, #FFFFFF);
}

/* ========== Weekday Headers ========== */
.weekday-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 1px solid var(--color-border, #E8DCCF);
}

.weekday {
  padding: 16px 12px;
  text-align: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-accent, #D4A373);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

/* ========== Days Grid ========== */
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.calendar-day {
  min-height: 130px;
  padding: 8px;
  border-bottom: 1px solid var(--color-border, #E8DCCF);
  border-right: 1px solid var(--color-border, #E8DCCF);
  background: var(--color-panel-bg, #FFFFFF);
  display: flex;
  flex-direction: column;
  transition: background-color 0.2s;
  animation: fadeIn 0.3s ease-out backwards;
}

.calendar-day:hover {
  background-color: var(--color-card-bg-hover, rgba(248, 245, 242, 0.5));
}

/* Remove right border for last column */
.calendar-day:nth-child(7n) {
  border-right: none;
}

/* Remove bottom border for last row */
.calendar-day:nth-child(n+36) {
  border-bottom: none;
}

/* ========== Other Month Days ========== */
.calendar-day.other-month {
  background: var(--color-card-bg, rgba(250, 248, 246, 0.5));
}

.calendar-day.other-month .day-number {
  color: var(--color-text-muted, rgba(140, 123, 112, 0.4));
}

/* ========== Today ========== */
.calendar-day.is-today {
  background: var(--color-panel-bg, #FFFFFF);
}

/* ========== Day Number ========== */
.day-number {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-main, #2C1810);
  padding: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
}

.day-number.today-badge {
  background: var(--color-secondary, #804030);
  color: var(--color-text-light, #FFFFFF);
  width: 28px;
  height: 28px;
  border-radius: 50%;
  box-shadow: 0 2px 8px rgba(var(--shadow-base, 128, 64, 48), 0.3);
}

/* ========== Events Container ========== */
.day-events {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  margin-top: 4px;
}

/* ========== Event Item ========== */
.event-item {
  display: block;
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 2px;
  border-left: 3px solid;
  font-weight: 500;
  line-height: 1.3;
  cursor: pointer;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  text-decoration: none;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.event-item:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md, 0 4px 12px rgba(128, 64, 48, 0.15));
}

/* Server Event Type - Primary color theme */
.event-item.server {
  background: var(--color-primary-light, rgba(128, 64, 48, 0.05));
  color: var(--color-secondary, #804030);
  border-color: var(--color-secondary, #804030);
}

.event-item.server:hover {
  background: var(--btn-secondary-bg, rgba(128, 64, 48, 0.1));
}

/* Guild Event Type - Accent/Gold theme */
.event-item.guild {
  background: rgba(var(--shadow-base, 212, 163, 115), 0.1);
  color: var(--color-primary, #8C5E35);
  border-color: var(--color-accent, #D4A373);
}

.event-item.guild:hover {
  background: rgba(var(--shadow-base, 212, 163, 115), 0.2);
}

.event-title {
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ========== More Events ========== */
.more-events {
  font-size: 10px;
  font-weight: 700;
  color: var(--color-text-secondary, #8C7B70);
  padding: 0 8px;
  margin-top: 2px;
  cursor: pointer;
  transition: color 0.2s;
}

.more-events:hover {
  color: var(--color-secondary, #804030);
}

/* ========== Fade In Animation ========== */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
