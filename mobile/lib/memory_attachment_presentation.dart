class MemoryAttachmentPresentation {
  const MemoryAttachmentPresentation({
    required this.body,
    required this.transcript,
    required this.content,
    required this.status,
    required this.errorMessage,
    required this.stateMessage,
    required this.isTranscript,
    required this.canRetry,
    required this.hasFilenamePlaceholder,
  });

  final String body;
  final String transcript;
  final String content;
  final String status;
  final String errorMessage;
  final String stateMessage;
  final bool isTranscript;
  final bool canRetry;
  final bool hasFilenamePlaceholder;
}

MemoryAttachmentPresentation resolveMemoryAttachmentPresentation({
  required String fileName,
  required String fileType,
  required String storedTranscript,
  required String storedContent,
  required String status,
  required String errorMessage,
  required String memoryTranscript,
  required int attachmentCount,
}) {
  final normalizedFileName = fileName.trim();
  final normalizedFileType = fileType.trim().toLowerCase();
  final normalizedTranscript = storedTranscript.trim();
  final normalizedContent = storedContent.trim();
  final normalizedStatus = status.trim().toLowerCase();
  final isTranscriptFile = const {
    'mp3',
    'wav',
    'm4a',
    'flac',
    'ogg',
    'mp4',
    'mov',
    'webm',
  }.contains(normalizedFileType);
  final filenamePlaceholder = isTranscriptFile &&
      normalizedTranscript.isEmpty &&
      (normalizedContent.isEmpty ||
          normalizedContent.toLowerCase() == normalizedFileName.toLowerCase());
  final inheritedTranscript = filenamePlaceholder && attachmentCount == 1
      ? memoryTranscript.trim()
      : '';
  final transcript = normalizedTranscript.isNotEmpty
      ? normalizedTranscript
      : inheritedTranscript;
  final content = filenamePlaceholder ? '' : normalizedContent;
  final body = transcript.isNotEmpty ? transcript : content;
  final unresolvedPlaceholder = filenamePlaceholder && transcript.isEmpty;
  final effectiveStatus =
      unresolvedPlaceholder && normalizedStatus == 'completed'
          ? 'failed'
          : normalizedStatus;
  final effectiveErrorMessage =
      unresolvedPlaceholder ? '音频未生成转写内容，请重新解析。' : errorMessage.trim();
  final stateMessage = switch (effectiveStatus) {
    'pending' => '文件已保存，等待解析。',
    'processing' || 'transcribing' => '正在解析文件内容，请稍候。',
    'skipped' =>
      effectiveErrorMessage.isNotEmpty ? effectiveErrorMessage : '该文件已跳过解析。',
    _ => effectiveErrorMessage.isNotEmpty
        ? effectiveErrorMessage
        : '文件解析失败，请重新解析。',
  };

  return MemoryAttachmentPresentation(
    body: body,
    transcript: transcript,
    content: content,
    status: effectiveStatus,
    errorMessage: effectiveErrorMessage,
    stateMessage: stateMessage,
    isTranscript: transcript.isNotEmpty || isTranscriptFile,
    canRetry: effectiveStatus == 'failed' || effectiveStatus == 'skipped',
    hasFilenamePlaceholder: filenamePlaceholder,
  );
}
