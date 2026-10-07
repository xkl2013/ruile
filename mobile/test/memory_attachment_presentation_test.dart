import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/memory_attachment_presentation.dart';

void main() {
  test('single audio attachment inherits the memory transcript', () {
    final presentation = resolveMemoryAttachmentPresentation(
      fileName: 'meeting.m4a',
      fileType: 'm4a',
      storedTranscript: '',
      storedContent: 'meeting.m4a',
      status: 'completed',
      errorMessage: '',
      memoryTranscript: '这是记忆级转写内容。',
      attachmentCount: 1,
    );

    expect(presentation.body, '这是记忆级转写内容。');
    expect(presentation.transcript, '这是记忆级转写内容。');
    expect(presentation.status, 'completed');
    expect(presentation.isTranscript, isTrue);
    expect(presentation.canRetry, isFalse);
  });

  test('completed audio filename placeholder becomes retryable failure', () {
    final presentation = resolveMemoryAttachmentPresentation(
      fileName: 'meeting.m4a',
      fileType: 'm4a',
      storedTranscript: '',
      storedContent: 'meeting.m4a',
      status: 'completed',
      errorMessage: '',
      memoryTranscript: '',
      attachmentCount: 1,
    );

    expect(presentation.body, isEmpty);
    expect(presentation.status, 'failed');
    expect(presentation.canRetry, isTrue);
    expect(presentation.errorMessage, '音频未生成转写内容，请重新解析。');
    expect(presentation.stateMessage, '音频未生成转写内容，请重新解析。');
  });

  test('pending audio placeholder keeps the waiting state', () {
    final presentation = resolveMemoryAttachmentPresentation(
      fileName: 'meeting.m4a',
      fileType: 'm4a',
      storedTranscript: '',
      storedContent: '',
      status: 'pending',
      errorMessage: '',
      memoryTranscript: '',
      attachmentCount: 1,
    );

    expect(presentation.status, 'pending');
    expect(presentation.canRetry, isFalse);
    expect(presentation.stateMessage, '文件已保存，等待解析。');
  });

  test('multiple audio attachments do not share a memory transcript', () {
    final presentation = resolveMemoryAttachmentPresentation(
      fileName: 'meeting.m4a',
      fileType: 'm4a',
      storedTranscript: '',
      storedContent: 'meeting.m4a',
      status: 'completed',
      errorMessage: '',
      memoryTranscript: '只能对应单个附件的记忆级转写。',
      attachmentCount: 2,
    );

    expect(presentation.body, isEmpty);
    expect(presentation.status, 'failed');
    expect(presentation.canRetry, isTrue);
  });

  test('parsed document content keeps its original presentation', () {
    final presentation = resolveMemoryAttachmentPresentation(
      fileName: 'plan.pdf',
      fileType: 'pdf',
      storedTranscript: '',
      storedContent: '# 项目计划',
      status: 'completed',
      errorMessage: '',
      memoryTranscript: '不应覆盖文档正文',
      attachmentCount: 1,
    );

    expect(presentation.body, '# 项目计划');
    expect(presentation.content, '# 项目计划');
    expect(presentation.status, 'completed');
    expect(presentation.isTranscript, isFalse);
    expect(presentation.canRetry, isFalse);
  });
}
