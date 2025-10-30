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

  it('renders title and subtitle', () => {
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

  it('shows provider label and status indicator', () => {
    const wrapper = mount(AIChatHeader, {
      props: {
        provider: 'deepseek',
        status: 'connecting'
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    const indicator = wrapper.find('.status-indicator')
    expect(indicator.exists()).toBe(true)
    expect(indicator.classes()).toContain('connecting')
    expect(indicator.text()).toContain('ai.status.connecting')
    expect(wrapper.text()).toContain('ai.providerLabel')
  })

  it('applies correct status classes', () => {
    const createWrapper = (status: 'connected' | 'connecting' | 'disconnected') => mount(AIChatHeader, {
      props: {
        provider: 'openai',
        status
      },
      global: {
        provide: {
          appContext: mockContext
        }
      }
    })

    expect(createWrapper('connected').find('.status-indicator').classes()).toContain('connected')
    expect(createWrapper('connecting').find('.status-indicator').classes()).toContain('connecting')
    expect(createWrapper('disconnected').find('.status-indicator').classes()).toContain('disconnected')
  })
})
