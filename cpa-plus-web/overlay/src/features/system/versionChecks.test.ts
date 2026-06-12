import { describe, expect, it } from 'vitest';
import { compareUpstreamVersions, readManagerLatestTag, readApiLatestVersion } from './versionChecks';

describe('versionChecks', () => {
  it('reads CPA-Manager-Plus latest commit SHA as short upstream version', () => {
    expect(readManagerLatestTag({ sha: '5ee6e23abcdef1234567890' })).toBe('5ee6e23a');
  });

  it('reads integrated Manager latest-version payload', () => {
    expect(readManagerLatestTag({ 'latest-version': 'v1.4.1+f2301ac8' })).toBe('v1.4.1+f2301ac8');
  });

  it('prefers integrated Manager latest-version over raw sha', () => {
    expect(
      readManagerLatestTag({
        'latest-version': 'v1.4.1+75e06189',
        sha: '75e06189faa06f61856f428b36d70971649393ca',
      })
    ).toBe('v1.4.1+75e06189');
  });

  it('reads CLIProxyAPI latest release version', () => {
    expect(readApiLatestVersion({ 'latest-version': 'v7.1.55' })).toBe('v7.1.55');
  });

  it('compares commit-like upstream versions by exact value', () => {
    expect(compareUpstreamVersions('5ee6e23a', '5ee6e23a')).toBe('latest');
    expect(compareUpstreamVersions('11111111', '5ee6e23a')).toBe('update');
  });

  it('treats same tag with different Plus commit suffix as an update', () => {
    expect(compareUpstreamVersions('v1.4.1+75e06189', 'v1.4.1+b4c93d13')).toBe('update');
  });

  it('treats tag+commit and matching plain commit as latest', () => {
    expect(compareUpstreamVersions('75e06189', 'v1.4.1+75e06189')).toBe('latest');
    expect(compareUpstreamVersions('v1.4.1+75e06189', '75e06189')).toBe('latest');
  });
});
