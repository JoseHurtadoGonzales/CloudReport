<template>
  <UApp>
    <!-- Immediate top-bar progress on every route change so the click always
         gets instant feedback, even before the page's data arrives. -->
    <NuxtLoadingIndicator color="#9fe870" :height="2.5" :duration="2500" :throttle="0" />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
