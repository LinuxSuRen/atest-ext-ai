import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import AIChatHeader from '@/components/AIChatHeader.vue'
import type { AppContext } from '@/types'

describe('AIChatHeader', () => {
  let mockContext: AppContext

  beforeEach(() => {
    mockContext = {
      i18n: {
        t: (key: string) => key,
        locale: { value: 'en' } as any
      },
      API: {},
      Cache: {}
    }
  })

  it('should render title and subtitle', () => {
    const wrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'disconnected'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    expect(wrapper.text()).toContain('ai.title')
    expect(wrapper.text()).toContain('ai.subtitle')
  })

  it('should display correct status', () => {
    const wrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'connected'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    expect(wrapper.text()).toContain('ai.status.connected')
  })

  it('should emit open-settings when settings button clicked', async () => {
    const wrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'disconnected'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('open-settings')).toBeTruthy()
    expect(wrapper.emitted('open-settings')?.length).toBe(1)
  })

  it('should display correct status type', () => {
    const connectedWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'connected'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    const connectingWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'connecting'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    const disconnectedWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'disconnected'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    // Status badges should exist in all cases
    expect(connectedWrapper.find('.el-tag').exists()).toBe(true)
    expect(connectingWrapper.find('.el-tag').exists()).toBe(true)
    expect(disconnectedWrapper.find('.el-tag').exists()).toBe(true)
  })
})
