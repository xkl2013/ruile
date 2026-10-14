import assert from 'node:assert/strict'
import test from 'node:test'
import type {
  ServiceFact,
  ServiceSpaceProfile,
  ServiceSubject,
} from '@/api/service'
import {
  buildCustomerInfo,
  displayValue,
  factToActivity,
} from './servicePresentation'

const profile = {
  id: 'profile-1',
  service_id: 'service-1',
  blueprint_version: 1,
  version: 2,
  schema: [
    {
      key: 'instruction_child',
      label: '孩子姓名',
      value_type: 'text',
      source: 'facts',
      required: false,
      sensitive: false,
      display_order: 1,
      aliases: ['学员姓名'],
    },
    {
      key: 'instruction_class',
      label: '班级',
      value_type: 'text',
      source: 'facts',
      required: false,
      sensitive: false,
      display_order: 2,
      aliases: ['所在班级'],
    },
    {
      key: 'instruction_teacher',
      label: '主班老师',
      value_type: 'text',
      source: 'facts',
      required: false,
      sensitive: false,
      display_order: 3,
    },
  ],
  values: {
    instruction_child: { value: '乐乐' },
    instruction_class: { value: '4班' },
    instruction_teacher: { value: '王老师' },
  },
} satisfies ServiceSpaceProfile

test('customer presentation reads instruction-derived profile fields by label', () => {
  const subject = {
    id: 'subject-1',
    tenant_id: 1,
    service_id: 'service-1',
    subject_type: 'member_family',
    subject_key: 'lele-family',
    display_name: '乐乐家庭',
  } satisfies ServiceSubject

  assert.deepEqual(buildCustomerInfo(subject, profile, '备用家庭'), {
    familyName: '乐乐家庭',
    student: '乐乐',
    className: '4班',
    teacher: '王老师',
  })
})

test('service profile values and activity use user-facing values and labels', () => {
  assert.equal(displayValue(profile.values.instruction_class), '4班')

  const activity = factToActivity({
    id: 'fact-1',
    tenant_id: 1,
    service_id: 'service-1',
    fact_type: 'profile_field',
    fact_key: 'instruction_class',
    value: { value: '4班' },
    source_type: 'chat_message',
    source_id: 'message-1',
  } satisfies ServiceFact, profile)

  assert.equal(activity.title, '班级')
  assert.equal(activity.detail, '4班')
})
