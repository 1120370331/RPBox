<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listEvents, type EventItem } from '@/api/post'

const router = useRouter()
const events = ref<EventItem[]>([])
const loading = ref(false)
const currentDate = ref(new Date())

// 当前年月
const currentYear = computed(() => currentDate.value.getFullYear())
const currentMonth = computed(() => currentDate.value.getMonth())

// 月份名称
const monthNames = ['一月', '二月', '三月', '四月', '五月', '六月',
                    '七月', '八月', '九月', '十月', '十一月', '十二月']

// 星期名称
const weekDays = ['日', '一', '二', '三', '四', '五', '六']

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

// 格式化时间
function formatTime(dateStr: string) {
  const date = new Date(dateStr)
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`
}

onMounted(() => {
  loadEvents()
})
</script>

<template>
  <div class="event-calendar">
    <div class="calendar-header">
      <div class="header-left">
        <h3 class="calendar-title">
          <i class="ri-calendar-event-line"></i>
          活动日历
        </h3>
      </div>
      <div class="header-center">
        <button class="nav-btn" @click="prevMonth">
          <i class="ri-arrow-left-s-line"></i>
        </button>
        <span class="current-month">{{ currentYear }}年 {{ monthNames[currentMonth] }}</span>
        <button class="nav-btn" @click="nextMonth">
          <i class="ri-arrow-right-s-line"></i>
        </button>
      </div>
      <div class="header-right">
        <button class="today-btn" @click="goToToday">今天</button>
      </div>
    </div>

    <div class="calendar-body">
      <div class="weekday-header">
        <div v-for="day in weekDays" :key="day" class="weekday">{{ day }}</div>
      </div>

      <div class="calendar-grid">
        <div
          v-for="(day, index) in calendarDays"
          :key="index"
          class="calendar-day"
          :class="{
            'other-month': !day.isCurrentMonth,
            'today': day.isCurrentMonth && isToday(day.date),
            'has-events': day.events.length > 0
          }"
        >
          <span class="day-number">{{ day.date }}</span>
          <div v-if="day.events.length > 0" class="day-events">
            <div
              v-for="event in day.events.slice(0, 2)"
              :key="event.id"
              class="event-dot"
              :class="event.event_type"
              :title="event.title"
              @click="viewEvent(event)"
            >
              <span class="event-time">{{ formatTime(event.event_start_time!) }}</span>
              <span class="event-title">{{ event.title }}</span>
            </div>
            <div v-if="day.events.length > 2" class="more-events">
              +{{ day.events.length - 2 }} 更多
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <i class="ri-loader-4-line spinning"></i>
    </div>
  </div>
</template>

<style scoped>
.event-calendar {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(75,54,33,0.05);
  position: relative;
}

.calendar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.calendar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  color: #4B3621;
  margin: 0;
}

.calendar-title i {
  font-size: 22px;
  color: #804030;
}

.header-center {
  display: flex;
  align-items: center;
  gap: 12px;
}

.nav-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: #F5EFE7;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
}

.nav-btn:hover {
  background: #E5D4C1;
}

.nav-btn i {
  font-size: 20px;
  color: #4B3621;
}

.current-month {
  font-size: 16px;
  font-weight: 600;
  color: #2C1810;
  min-width: 120px;
  text-align: center;
}

.today-btn {
  padding: 6px 16px;
  border: 2px solid #804030;
  background: #fff;
  color: #804030;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.today-btn:hover {
  background: #804030;
  color: #fff;
}

.calendar-body {
  border: 1px solid #E5D4C1;
  border-radius: 12px;
  overflow: hidden;
}

.weekday-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  background: #F5EFE7;
}

.weekday {
  padding: 12px;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: #4B3621;
}

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.calendar-day {
  min-height: 80px;
  padding: 8px;
  border-top: 1px solid #E5D4C1;
  border-right: 1px solid #E5D4C1;
  background: #fff;
  transition: background 0.2s;
}

.calendar-day:nth-child(7n) {
  border-right: none;
}

.calendar-day.other-month {
  background: #FAFAFA;
}

.calendar-day.other-month .day-number {
  color: #CCC;
}

.calendar-day.today {
  background: #FFF9F0;
}

.calendar-day.today .day-number {
  background: #804030;
  color: #fff;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.calendar-day.has-events:hover {
  background: #FFF5E6;
}

.day-number {
  font-size: 14px;
  color: #2C1810;
  font-weight: 500;
}

.day-events {
  margin-top: 4px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.event-dot {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  transition: all 0.2s;
}

.event-dot.server {
  background: #E3F2FD;
  color: #1565C0;
}

.event-dot.guild {
  background: #FFF3E0;
  color: #E65100;
}

.event-dot:hover {
  transform: scale(1.02);
}

.event-time {
  font-weight: 600;
  flex-shrink: 0;
}

.event-title {
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-events {
  font-size: 11px;
  color: #8D7B68;
  padding: 2px 6px;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255,255,255,0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
}

.spinning {
  font-size: 32px;
  color: #804030;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
