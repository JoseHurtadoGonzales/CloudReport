<script setup lang="ts">
import * as z from 'zod'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: false })

const auth = useAuthStore()
const { t } = useI18n()

const schema = computed(() => z.object({
  username: z.string().min(1, t('login.errUsername')),
  password: z.string().min(1, t('login.errPassword')),
}))

const username = ref('')
const password = ref('')
const remember = ref(true)
const showPassword = ref(false)
const errors = reactive<{ username?: string; password?: string }>({})

onMounted(() => {
  auth.hydrate()
  if (auth.isAuthenticated) navigateTo('/')
})

async function onSubmit(e: Event) {
  e.preventDefault()
  if (auth.loading) return

  const parsed = schema.value.safeParse({ username: username.value, password: password.value })
  errors.username = undefined
  errors.password = undefined
  if (!parsed.success) {
    for (const err of parsed.error.issues) {
      const key = err.path[0] as 'username' | 'password'
      errors[key] = err.message
    }
    return
  }

  const ok = await auth.login(username.value.trim(), password.value)
  if (ok) await navigateTo('/')
}

const features = computed(() => [
  t('login.featPaged'),
  t('login.featDocx'),
  t('login.featXlsx'),
  t('login.featScopes'),
  t('login.featOdata'),
  t('login.featWebhooks'),
])
</script>

<template>
  <div class="relative min-h-dvh w-full flex items-center justify-center px-4 py-12">
    <AnimatedBackdrop />

    <!-- Brand bar -->
    <header class="absolute top-5 left-5 right-5 sm:top-6 sm:left-8 sm:right-8 flex items-center justify-between z-10 cr-anim-fade-up">
      <NuxtLink to="/" class="flex items-center gap-2.5 group">
        <BrandLogo :size="36" />
        <span class="font-bold text-[15px] tracking-tight" style="color: var(--cr-text)">
          cloud-report
        </span>
      </NuxtLink>

      <div class="flex items-center gap-2">
        <a
          href="https://github.com/jsreport/jsreport"
          target="_blank"
          rel="noopener"
          class="hidden sm:inline-flex items-center gap-1 px-3 py-2 rounded-full text-[13px] font-semibold transition-colors duration-150"
          style="color: var(--cr-text-muted)"
        >
          {{ t('login.docs') }}
          <UIcon name="i-lucide-arrow-up-right" class="w-3.5 h-3.5" />
        </a>
        <ColorModeToggle />
      </div>
    </header>

    <!-- 2-column grid: hero + form -->
    <div class="relative z-10 w-full max-w-6xl grid lg:grid-cols-[1.05fr_1fr] gap-10 lg:gap-20 items-center">
      <!-- LEFT: marketing hero (≥ lg) -->
      <div class="hidden lg:flex flex-col gap-7 cr-stagger">
        <span class="cr-badge-prod self-start">
          <span class="cr-badge-prod-dot cr-anim-pulse-dot" />
          {{ t('login.apiInProd') }}
        </span>

        <h1
          class="text-[56px] xl:text-[68px] leading-[0.95] font-black tracking-tight text-balance"
          style="color: var(--cr-text)"
        >
          {{ t('login.h1Part1') }}
          <br />
          <span class="bg-clip-text text-transparent" style="background-image: linear-gradient(90deg, var(--color-wise-700) 0%, var(--color-wise-500) 100%)">
            {{ t('login.h1Part2') }}
          </span>
          {{ t('login.h1Part3') }}
        </h1>

        <p class="text-[17px] max-w-md leading-relaxed" style="color: var(--cr-text-muted)">
          {{ t('login.heroDesc') }}
        </p>

        <ul class="grid grid-cols-2 gap-x-6 gap-y-3 mt-1 max-w-md">
          <li
            v-for="item in features"
            :key="item"
            class="flex items-center gap-2.5 text-[14px] font-medium"
            style="color: var(--cr-text)"
          >
            <span
              class="w-5 h-5 rounded-full flex items-center justify-center shrink-0"
              style="background: var(--color-wise-400)"
            >
              <UIcon name="i-lucide-check" class="w-3 h-3 text-[#0e0f0c]" />
            </span>
            {{ item }}
          </li>
        </ul>

        <!-- Decorative mini-card peek -->
        <div
          class="hidden xl:flex items-center gap-3 mt-3 p-4 cr-card max-w-md"
          style="border-radius: 16px"
        >
          <div class="w-10 h-10 rounded-md bg-wise-100 flex items-center justify-center shrink-0">
            <UIcon name="i-lucide-zap" class="w-5 h-5 text-wise-700" />
          </div>
          <div class="text-[13px]" style="color: var(--cr-text-muted)">
            <div class="font-semibold mb-0.5" style="color: var(--cr-text)">
              {{ t('login.fastRender') }}
            </div>
            {{ t('login.fastRenderDescPre') }} <span class="font-semibold tabular-nums" style="color: var(--cr-text)">120 ms</span> {{ t('login.fastRenderDescMid') }} <span class="font-semibold tabular-nums" style="color: var(--cr-text)">800 ms</span> {{ t('login.fastRenderDescEnd') }}
          </div>
        </div>
      </div>

      <!-- RIGHT: login card -->
      <div class="w-full max-w-md mx-auto cr-anim-fade-up" style="animation-delay: 60ms">
        <div class="cr-card p-8 sm:p-10">
          <div class="cr-stagger">
            <p class="cr-eyebrow mb-2" style="color: var(--color-positive)">
              {{ t('login.welcomeBack') }}
            </p>
            <h2 class="text-[26px] font-bold tracking-tight" style="color: var(--cr-text)">
              {{ t('login.title') }}
            </h2>
            <p class="text-[14px] mt-1.5" style="color: var(--cr-text-muted)">
              {{ t('login.subtitle') }}
            </p>
          </div>

          <form class="mt-7 space-y-5" novalidate @submit="onSubmit">
            <!-- Username -->
            <div class="cr-stagger">
              <label for="cr-username" class="cr-label">{{ t('login.usernameLabel') }}</label>
              <div class="relative">
                <UIcon name="i-lucide-user" class="cr-input-icon w-5 h-5" />
                <input
                  id="cr-username"
                  v-model="username"
                  type="text"
                  autocomplete="username"
                  required
                  :placeholder="t('login.usernamePlaceholder')"
                  class="cr-input"
                  :class="errors.username ? 'cr-input--invalid' : ''"
                  :aria-invalid="!!errors.username"
                  :aria-describedby="errors.username ? 'cr-username-error' : undefined"
                />
              </div>
              <Transition
                enter-active-class="transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
                leave-active-class="transition-all duration-150"
                enter-from-class="opacity-0 -translate-y-1"
                leave-to-class="opacity-0"
              >
                <p
                  v-if="errors.username"
                  id="cr-username-error"
                  class="mt-1.5 text-[12px] font-medium flex items-center gap-1.5"
                  style="color: #d03238"
                >
                  <UIcon name="i-lucide-circle-alert" class="w-3.5 h-3.5" />
                  {{ errors.username }}
                </p>
              </Transition>
            </div>

            <!-- Password -->
            <div>
              <label for="cr-password" class="cr-label">{{ t('login.passwordLabel') }}</label>
              <div class="relative">
                <UIcon name="i-lucide-lock" class="cr-input-icon w-5 h-5" />
                <input
                  id="cr-password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  required
                  placeholder="••••••••"
                  class="cr-input"
                  :class="errors.password ? 'cr-input--invalid' : ''"
                  :aria-invalid="!!errors.password"
                  :aria-describedby="errors.password ? 'cr-password-error' : undefined"
                />
                <button
                  type="button"
                  class="cr-input-affix"
                  :aria-label="showPassword ? t('login.hidePassword') : t('login.showPassword')"
                  @click="showPassword = !showPassword"
                >
                  <UIcon :name="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="w-4 h-4" />
                </button>
              </div>
              <Transition
                enter-active-class="transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
                leave-active-class="transition-all duration-150"
                enter-from-class="opacity-0 -translate-y-1"
                leave-to-class="opacity-0"
              >
                <p
                  v-if="errors.password"
                  id="cr-password-error"
                  class="mt-1.5 text-[12px] font-medium flex items-center gap-1.5"
                  style="color: #d03238"
                >
                  <UIcon name="i-lucide-circle-alert" class="w-3.5 h-3.5" />
                  {{ errors.password }}
                </p>
              </Transition>
            </div>

            <!-- Remember row -->
            <div class="flex items-center">
              <label class="inline-flex items-center gap-2.5 cursor-pointer select-none">
                <input v-model="remember" type="checkbox" class="cr-checkbox" />
                <span class="text-[13.5px]" style="color: var(--cr-text-muted)">
                  {{ t('login.rememberMe') }}
                </span>
              </label>
            </div>

            <!-- Inline error from backend -->
            <Transition
              enter-active-class="transition-all duration-250 ease-[cubic-bezier(0.22,1,0.36,1)]"
              leave-active-class="transition-all duration-150"
              enter-from-class="opacity-0 -translate-y-2"
              leave-to-class="opacity-0"
            >
              <div
                v-if="auth.error"
                role="alert"
                class="rounded-xl px-4 py-3 flex items-start gap-2.5 text-[13px]"
                style="background: rgb(208 50 56 / 0.08); color: #a72027; border: 1px solid rgb(208 50 56 / 0.20)"
              >
                <UIcon name="i-lucide-circle-alert" class="w-4 h-4 mt-0.5 shrink-0" />
                <span>{{ auth.error }}</span>
              </div>
            </Transition>

            <!-- Submit -->
            <div>
              <button type="submit" class="cr-btn-primary" :disabled="auth.loading">
                <template v-if="!auth.loading">
                  <span>{{ t('login.submitArrow') }}</span>
                  <UIcon name="i-lucide-arrow-right" class="w-4 h-4" />
                </template>
                <template v-else>
                  <span class="inline-block w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: #0e0f0c; border-top-color: transparent" />
                  <span>{{ t('common.verifying') }}</span>
                </template>
                <span
                  v-if="auth.loading"
                  class="absolute inset-y-0 left-0 w-1/2 cr-anim-shimmer pointer-events-none"
                  style="background: linear-gradient(90deg, transparent, rgb(255 255 255 / 0.5), transparent)"
                />
              </button>
            </div>
          </form>

        </div>
      </div>
    </div>

    <!-- Footer -->
    <p
      class="absolute bottom-5 left-0 right-0 text-center cr-eyebrow"
      style="color: var(--cr-text-soft)"
    >
      cloud-report · v1.0.0 · {{ new Date().getFullYear() }}
    </p>
  </div>
</template>
