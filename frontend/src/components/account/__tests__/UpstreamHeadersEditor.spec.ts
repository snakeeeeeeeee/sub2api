import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import UpstreamHeadersEditor from '../UpstreamHeadersEditor.vue'

const iconStub = {
  template: '<span />'
}

describe('UpstreamHeadersEditor', () => {
  it('emits upstream header templates from edited rows', async () => {
    const wrapper = mount(UpstreamHeadersEditor, {
      global: {
        stubs: { Icon: iconStub }
      }
    })

    await wrapper.get('[data-testid="upstream-headers-toggle"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-add"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-name"]').setValue('X-Pool-Session-ID')
    await wrapper.get('[data-testid="upstream-header-value"]').setValue('{{header.session-id}}')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual({
      'X-Pool-Session-ID': '{{header.session-id}}'
    })
  })

  it('keeps a blank custom row visible while the parent v-model is still empty', async () => {
    const Parent = defineComponent({
      components: { UpstreamHeadersEditor },
      setup() {
        return { headers: ref<Record<string, string>>({}) }
      },
      template: '<UpstreamHeadersEditor v-model="headers" />'
    })
    const wrapper = mount(Parent, {
      global: {
        stubs: { Icon: iconStub }
      }
    })

    await wrapper.get('[data-testid="upstream-headers-toggle"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-add"]').trigger('click')

    expect(wrapper.findAll('[data-testid="upstream-header-row"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="upstream-header-name"]').element).toHaveProperty('value', '')
    expect(wrapper.get('[data-testid="upstream-header-value"]').element).toHaveProperty('value', '')
  })

  it('allows arbitrary header names and fixed template values', async () => {
    const wrapper = mount(UpstreamHeadersEditor, {
      global: {
        stubs: { Icon: iconStub }
      }
    })

    await wrapper.get('[data-testid="upstream-headers-toggle"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-add"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-name"]').setValue('X-Custom-Pool-Token')
    await wrapper
      .get('[data-testid="upstream-header-value"]')
      .setValue('fixed-{{query.pool_id}}-{{account.extra.routing.tenant}}')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual({
      'X-Custom-Pool-Token': 'fixed-{{query.pool_id}}-{{account.extra.routing.tenant}}'
    })
  })

  it('inserts variables into the free-form value input', async () => {
    const wrapper = mount(UpstreamHeadersEditor, {
      global: {
        stubs: { Icon: iconStub }
      }
    })

    await wrapper.get('[data-testid="upstream-headers-toggle"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-add"]').trigger('click')
    await wrapper.get('[data-testid="upstream-header-name"]').setValue('X-Session')
    await wrapper.get('[data-testid="upstream-header-value"]').setValue('prefix-')
    await wrapper.get('[data-testid="upstream-header-variable"]').setValue('{{header.session-id}}')

    expect(wrapper.get('[data-testid="upstream-header-value"]').element).toHaveProperty(
      'value',
      'prefix-{{header.session-id}}'
    )
    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual({
      'X-Session': 'prefix-{{header.session-id}}'
    })
  })

  it('renders existing config and clears when last row is deleted', async () => {
    const wrapper = mount(UpstreamHeadersEditor, {
      props: {
        modelValue: {
          'X-Tenant-ID': '{{body.metadata.tenant_id}}'
        }
      },
      global: {
        stubs: { Icon: iconStub }
      }
    })

    expect(wrapper.get('[data-testid="upstream-header-name"]').element).toHaveProperty(
      'value',
      'X-Tenant-ID'
    )
    expect(wrapper.get('[data-testid="upstream-header-value"]').element).toHaveProperty(
      'value',
      '{{body.metadata.tenant_id}}'
    )

    await wrapper.get('[data-testid="upstream-header-remove"]').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual({})
  })

  it('can add preset rows', async () => {
    const wrapper = mount(UpstreamHeadersEditor, {
      global: {
        stubs: { Icon: iconStub }
      }
    })

    await wrapper.get('[data-testid="upstream-headers-toggle"]').trigger('click')
    const presetButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('X-Account-ID'))
    expect(presetButton).toBeTruthy()
    await presetButton!.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual({
      'X-Account-ID': '{{account.id}}'
    })
  })
})
