import type { ManagerLatestRelease } from '@/services/api/version';
import { compareVersions } from '@/utils/version';

type VersionPayload = Record<string, unknown> | undefined | null;

export type UpstreamVersionState = 'unknown' | 'latest' | 'update' | 'older';

export const readManagerLatestTag = (data: ManagerLatestRelease | VersionPayload): string => {
  if (!data) return '';
  const raw =
    data.sha ??
    data['latest-commit'] ??
    data.latest_commit ??
    data.tag_name ??
    data.name ??
    data['latest-version'] ??
    data.latest_version ??
    data.latest;
  const value = typeof raw === 'string' ? raw : raw == null ? '' : String(raw);
  return value.length > 8 && /^[0-9a-f]{9,}$/i.test(value) ? value.slice(0, 8) : value;
};

export const readApiLatestVersion = (data: VersionPayload): string => {
  if (!data) return '';
  const raw = data['latest-version'] ?? data.latest_version ?? data.latest;
  return typeof raw === 'string' ? raw : raw == null ? '' : String(raw);
};

export const compareUpstreamVersions = (latest?: string | null, current?: string | null): UpstreamVersionState => {
  const normalizedLatest = (latest || '').trim();
  const normalizedCurrent = (current || '').trim();
  if (!normalizedLatest || !normalizedCurrent) return 'unknown';
  if (normalizedLatest === normalizedCurrent) return 'latest';
  if (/[+]([0-9a-f]{7,})$/i.test(normalizedLatest) || /[+]([0-9a-f]{7,})$/i.test(normalizedCurrent)) {
    return 'update';
  }

  const semanticComparison = compareVersions(normalizedLatest, normalizedCurrent);
  if (semanticComparison === 0) return 'latest';
  if (semanticComparison === 1) return 'update';
  if (semanticComparison === -1) return 'older';

  return 'update';
};
