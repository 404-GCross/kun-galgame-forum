<script setup lang="ts">
import { kunSubSites } from '~/constants/layout'

const utmLink = useUtmLink()
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      name="子网站"
      description="鲲 Galgame 旗下的其他站点与资源, 均为我们自己维护, 欢迎逛逛~"
    />

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <KunCard
        v-for="site in kunSubSites"
        :key="site.link"
        :is-transparent="false"
        :is-hoverable="true"
        content-class="flex h-full flex-col gap-3"
      >
        <a
          :href="utmLink(site.link)"
          target="_blank"
          rel="noopener noreferrer"
          class="group flex grow flex-col gap-2"
        >
          <div class="flex items-center gap-2">
            <KunIcon :name="site.icon" class="text-primary text-2xl" />
            <h3
              class="group-hover:text-primary text-lg font-bold transition-colors"
            >
              {{ site.name }}
            </h3>
            <KunChip v-if="site.hint" color="primary" size="sm">
              {{ site.hint }}
            </KunChip>
          </div>
          <p class="text-default-500 grow text-sm leading-6">
            {{ site.description }}
          </p>
        </a>

        <KunLink
          v-if="site.github"
          :to="site.github"
          target="_blank"
          rel="noopener noreferrer"
          color="primary"
          size="sm"
          underline="hover"
        >
          <KunIcon name="lucide:github" class="text-base" />
          GitHub 开源地址
        </KunLink>
      </KunCard>
    </div>
  </div>
</template>
