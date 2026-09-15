import 'dart:convert';

import 'recording_card_support.dart';

const int jieliRecordingCardTotalStorageBytes = 64 * 1024 * 1024 * 1024;

bool isJieliRecordingAudioFileName(String name) {
  return name.trim().toLowerCase().endsWith('.mp3');
}

bool isJieliRecordingSidecarFileName(String name) {
  return name.trim().toLowerCase().endsWith('.txt');
}

String jieliFileNameNoExtension(String name) {
  final trimmed = name.trim();
  if (trimmed.isEmpty) return '';
  final dot = trimmed.lastIndexOf('.');
  if (dot <= 0) return trimmed;
  return trimmed.substring(0, dot);
}

String jieliSidecarFileNameFor(String fileNameNoExt) {
  final normalized = jieliFileNameNoExtension(fileNameNoExt);
  return normalized.isEmpty ? '' : '$normalized.TXT';
}

bool shouldShowJieliRecordingFile(RecordingCardFileEntry entry) {
  return entry.transferStatus != RecordingCardFileTransferStatus.synced &&
      entry.transferStatus != RecordingCardFileTransferStatus.deletedOnDevice;
}

String formatJieliRecordingCardRemainingStorage(int? usedBytes) {
  if (usedBytes == null || usedBytes < 0) return '--';

  final remainingBytes = (jieliRecordingCardTotalStorageBytes - usedBytes)
      .clamp(0, jieliRecordingCardTotalStorageBytes)
      .toInt();
  if (remainingBytes == 0) return '0G';

  final gigabytes = remainingBytes / (1024 * 1024 * 1024);
  final fixed = gigabytes.toStringAsFixed(1);
  return '${fixed.endsWith('.0') ? fixed.substring(0, fixed.length - 2) : fixed}G';
}

RecordingCardFileTransferStatus jieliStatusAfterFileSeen(
  RecordingCardFileEntry? existing,
) {
  if (existing == null) return RecordingCardFileTransferStatus.downloadPending;
  final status = existing.transferStatus;
  if (status == RecordingCardFileTransferStatus.deletedOnDevice) {
    return existing.cloudMemoryId.trim().isNotEmpty
        ? RecordingCardFileTransferStatus.synced
        : RecordingCardFileTransferStatus.downloadPending;
  }
  if (status == RecordingCardFileTransferStatus.downloading ||
      status == RecordingCardFileTransferStatus.retryPending ||
      status == RecordingCardFileTransferStatus.checksumFailed ||
      status == RecordingCardFileTransferStatus.failed) {
    return RecordingCardFileTransferStatus.downloadPending;
  }
  if (status == RecordingCardFileTransferStatus.cloudSyncing) {
    return RecordingCardFileTransferStatus.cloudSyncPending;
  }
  if (status == RecordingCardFileTransferStatus.cloudSyncFailed &&
      existing.localSbcPath.trim().isEmpty) {
    return RecordingCardFileTransferStatus.downloadPending;
  }
  return status;
}

RecordingCardFileEntry normalizeJieliRestoredAutoSyncEntry(
  RecordingCardFileEntry entry,
) {
  if (entry.transferStatus != RecordingCardFileTransferStatus.cloudSyncing) {
    return entry;
  }
  return entry.copyWith(
    transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
  );
}

RecordingCardFileEntry? nextJieliDownloadCandidate(
  Iterable<RecordingCardFileEntry> entries,
  Set<String> availableAudioFileNames,
) {
  final candidates = entries.where((entry) {
    if (!availableAudioFileNames.contains(entry.fileNameNoExt)) return false;
    return entry.transferStatus == RecordingCardFileTransferStatus.listed ||
        entry.transferStatus ==
            RecordingCardFileTransferStatus.downloadPending ||
        entry.transferStatus == RecordingCardFileTransferStatus.retryPending ||
        entry.transferStatus == RecordingCardFileTransferStatus.failed ||
        entry.transferStatus == RecordingCardFileTransferStatus.checksumFailed;
  }).toList()
    ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
  return candidates.isEmpty ? null : candidates.first;
}

RecordingCardFileEntry? nextJieliCloudCandidate(
  Iterable<RecordingCardFileEntry> entries,
) {
  final candidates = entries.where((entry) {
    if (entry.transferStatus ==
            RecordingCardFileTransferStatus.cloudSyncFailed &&
        entry.localSbcPath.trim().isEmpty) {
      return false;
    }
    return entry.transferStatus == RecordingCardFileTransferStatus.downloaded ||
        entry.transferStatus ==
            RecordingCardFileTransferStatus.cloudSyncPending ||
        entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncFailed;
  }).toList()
    ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
  return candidates.isEmpty ? null : candidates.first;
}

RecordingCardFileEntry? nextJieliDeleteCandidate(
  Iterable<RecordingCardFileEntry> entries,
  Set<String> availableAudioFileNames,
) {
  final candidates = entries.where((entry) {
    return entry.transferStatus == RecordingCardFileTransferStatus.synced &&
        entry.cloudMemoryId.trim().isNotEmpty &&
        availableAudioFileNames.contains(entry.fileNameNoExt);
  }).toList()
    ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
  return candidates.isEmpty ? null : candidates.first;
}

String? nextJieliSidecarDeleteCandidate(
  Iterable<RecordingCardFileEntry> entries,
  Set<String> availableSidecarFileNames,
  Set<String> availableAudioFileNames,
) {
  final candidates = entries.where((entry) {
    if (entry.cloudMemoryId.trim().isEmpty) return false;
    if (!availableSidecarFileNames.contains(entry.fileNameNoExt)) return false;
    if (availableAudioFileNames.contains(entry.fileNameNoExt)) return false;
    return entry.transferStatus == RecordingCardFileTransferStatus.synced ||
        entry.transferStatus == RecordingCardFileTransferStatus.deletedOnDevice;
  }).toList()
    ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
  return candidates.isEmpty ? null : candidates.first.fileNameNoExt;
}

String decodeJieliSidecarText(List<int> bytes) {
  if (bytes.isEmpty) return '';
  final decoded = utf8.decode(bytes, allowMalformed: true);
  final buffer = StringBuffer();
  for (final rune in decoded.runes) {
    if (rune == 0) continue;
    if (rune < 0x20 && rune != 0x09 && rune != 0x0a && rune != 0x0d) {
      continue;
    }
    buffer.writeCharCode(rune);
  }
  return buffer.toString().trim();
}

DateTime? parseJieliSidecarRecordedAt(String text) {
  final match = RegExp(
    r'^\s*(\d{4})-(\d{2})-(\d{2})-(\d{2})-(\d{2})\s*$',
  ).firstMatch(text);
  if (match == null) return null;
  final year = int.tryParse(match.group(1)!);
  final month = int.tryParse(match.group(2)!);
  final day = int.tryParse(match.group(3)!);
  final hour = int.tryParse(match.group(4)!);
  final minute = int.tryParse(match.group(5)!);
  if (year == null ||
      month == null ||
      day == null ||
      hour == null ||
      minute == null) {
    return null;
  }
  final parsed = DateTime(year, month, day, hour, minute);
  if (parsed.year != year ||
      parsed.month != month ||
      parsed.day != day ||
      parsed.hour != hour ||
      parsed.minute != minute) {
    return null;
  }
  return parsed;
}

String jieliTextPreview(
  String text, {
  int maxChars = 4000,
}) {
  final normalized = text.trim();
  if (normalized.length <= maxChars) return normalized;
  return '${normalized.substring(0, maxChars)}...';
}
