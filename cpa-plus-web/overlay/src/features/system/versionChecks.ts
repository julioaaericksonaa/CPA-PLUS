import type { ManagerLatestRelease } from '@/services/api/version';
import { compareVersions } from '@/utils/version';

type VersionPayload = Record<string, unknown> | undefined | null;

export type UpstreamVersionState = 'unknown' | 'latest' | 'update' | 'older';

export const readManagerLatestTag = (data: ManagerLatestRelease | VersionPayload): string => {
  if (!data) return '';
  const raw =
    data['latest-version'] ??
    data.latest_version ??
    data.latest ??
    data.tag_name ??
    data.name ??
    data.sha ??
    data['latest-commit'] ??
    data.latest_commit;
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

  const latestCommit = extractCommitSuffix(normalizedLatest);
  const currentCommit = extractCommitSuffix(normalizedCurrent);
  if (latestCommit && currentCommit) {
    return sameCommit(latestCommit, currentCommit) ? 'latest' : 'update';
  }
  if (latestCommit || currentCommit) {
    return 'update';
  }

  const semanticComparison = compareVersions(normalizedLatest, normalizedCurrent);
  if (semanticComparison === 0) return 'latest';
  if (semanticComparison === 1) return 'update';
  if (semanticComparison === -1) return 'older';

  return 'update';
};

const extractCommitSuffix = (value: string): string => {
  const commitWithVersion = value.match(/[+]([0-9a-f]{7,40})$/i)?.[1];
  if (commitWithVersion) return commitWithVersion.toLowerCase();
  const pureCommit = value.match(/^[0-9a-f]{7,40}$/i)?.[0];
  return pureCommit ? pureCommit.toLowerCase() : '';
};

const sameCommit = (left: string, right: string): boolean =>
  left === right || left.startsWith(right) || right.startsWith(left);
