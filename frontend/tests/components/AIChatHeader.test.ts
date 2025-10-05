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
        },
        stubs: {
          'el-tag': true,
          'el-icon': true,
          'el-button': true
        }
      }
    })

    // Find the el-button component and trigger its click event
    await wrapper.findComponent({ name: 'el-button' }).trigger('click')

    expect(wrapper.emitted('open-settings')).toBeTruthy()
    expect(wrapper.emitted('open-settings')?.length).toBe(1)
  })

  it('should display correct status type', () => {
    const globalConfig = {
      provide: {
        appContext: mockContext
      },
      stubs: {
        'el-tag': {
          template: '<span class="el-tag"><slot /></span>'
        },
        'el-icon': true,
        'el-button': true
      }
    }

    const connectedWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'connected'
      },
      global: globalConfig
    })

    const connectingWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'connecting'
      },
      global: globalConfig
    })

    const disconnectedWrapper = mount(AIChatHeader, {
      props: {
        provider: 'ollama',
        status: 'disconnected'
      },
      global: globalConfig
    })

    // Status badges should exist in all cases
    expect(connectedWrapper.find('.el-tag').exists()).toBe(true)
    expect(connectingWrapper.find('.el-tag').exists()).toBe(true)
    expect(disconnectedWrapper.find('.el-tag').exists()).toBe(true)
  })
})
