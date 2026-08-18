package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateKnowledgeBaseIconUsesCompactImageReferences(t *testing.T) {
	require.NoError(t, validateKnowledgeBaseIcon(""))
	require.NoError(t, validateKnowledgeBaseIcon("folder"))
	require.NoError(t, validateKnowledgeBaseIcon("image:resource://abcdefghijklmnopqrstu1"))
	require.NoError(t, validateKnowledgeBaseIcon("image:oss://bucket/path/icon.png"))

	require.Error(t, validateKnowledgeBaseIcon("image:data:image/png;base64,abc"))
	require.Error(t, validateKnowledgeBaseIcon(strings.Repeat("x", maxKnowledgeBaseIconBytes+1)))
}

func TestKnowledgeBaseIconImageExtSniffsImageData(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	ext, err := knowledgeBaseIconImageExt(png)
	require.NoError(t, err)
	require.Equal(t, ".png", ext)

	webp := []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}
	ext, err = knowledgeBaseIconImageExt(webp)
	require.NoError(t, err)
	require.Equal(t, ".webp", ext)

	_, err = knowledgeBaseIconImageExt([]byte("not an image"))
	require.Error(t, err)
}
