/**
 * 版本相关 API
 */

import { apiClient } from './client';

export interface ManagerLatestRelease {
  sha?: string;
  tag_name?: string;
  name?: string;
  html_url?: string;
  published_at?: string;
  [key: string]: unknown;
}

export const versionApi = {
  checkLatest: () => apiClient.get<Record<string, unknown>>('/latest-version'),

  checkManagerLatest: () => apiClient.get<ManagerLatestRelease>('/manager-latest-version'),

  triggerCPAPlusUpdate: () => apiClient.post<Record<string, unknown>>('/cpa-plus/update')
};
