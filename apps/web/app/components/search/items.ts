interface NavItem {
  textValue: string
  value: SearchType
}

export const navItems: NavItem[] = [
  {
    textValue: '话题',
    value: 'topic'
  },
  {
    textValue: 'Galgame',
    value: 'galgame'
  },
  {
    textValue: 'Galgame 工具',
    value: 'toolset'
  },
  {
    textValue: '用户',
    value: 'user'
  },
  {
    textValue: '回复',
    value: 'reply'
  },
  {
    textValue: '评论',
    value: 'comment'
  }
]
