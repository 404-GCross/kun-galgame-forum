<script setup lang="ts">
defineProps<{
  user: UserInfo
}>()
</script>

<template>
  <div class="h-full w-full space-y-6">
    <UserSettingAvatar />
    <UserSettingBio :user="user" />
    <UserSettingUsername />
    <UserSettingEmail />
    <UserSettingPassword />
    <UserSettingNotification />
  </div>
</template>
