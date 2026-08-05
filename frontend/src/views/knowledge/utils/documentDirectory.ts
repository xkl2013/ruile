const normalizePath = (value: string) => value
  .replace(/\\/g, '/')
  .split('/')
  .map(part => part.trim())
  .filter(part => part && part !== '.' && part !== '..');

/**
 * Builds the display path persisted with an uploaded knowledge document.
 * Folder uploads retain their relative hierarchy beneath the selected folder.
 */
export const getUploadDisplayFileName = (
  fileName: string,
  activeDirectoryPath: string,
  relativePath?: string,
) => {
  const directoryParts = normalizePath(activeDirectoryPath);
  const relativeParts = relativePath ? normalizePath(relativePath) : [];

  // webkitRelativePath starts with the selected local folder name. It is a
  // browser-only source label, so omit it from the knowledge-base directory.
  const nestedParts = relativeParts.length > 2
    ? relativeParts.slice(1, -1)
    : [];
  const name = normalizePath(fileName).pop() || fileName;

  return [...directoryParts, ...nestedParts, name].join('/');
};

/**
 * Builds a directory-aware display path for a web import without changing the
 * source URL. The path is only needed outside the root directory.
 */
export const getURLDisplayPath = (url: string, activeDirectoryPath: string) => {
  const directoryParts = normalizePath(activeDirectoryPath);
  if (directoryParts.length === 0) return '';

  let name = 'webpage';
  try {
    const parsed = new URL(url);
    const decodedPathname = decodeURIComponent(parsed.pathname);
    name = normalizePath(decodedPathname).pop() || parsed.hostname || name;
  } catch {
    // The URL input validates before import; keep a safe fallback for callers.
  }

  return [...directoryParts, name].join('/');
};
