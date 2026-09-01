<script setup lang="ts">
import { computed } from "vue"

const props = defineProps(['reviewImg', 'reviewUrl'])
const emit = defineEmits(['closePreImg'])
const close = () => {
    emit('closePreImg')
}
const previewImages = computed(() => {
    const url = typeof props.reviewUrl === 'string' ? props.reviewUrl.trim() : ''
    return url ? [url] : []
})
const shouldShowViewer = computed(() => Boolean(props.reviewImg && previewImages.value.length))
</script>
<template>
    <t-image-viewer v-if="shouldShowViewer" :visible="reviewImg" closeOnOverlay closeOnEscKeydown @close="close"
        :images="previewImages">
    </t-image-viewer>
</template>
<style scoped lang="less"></style>
