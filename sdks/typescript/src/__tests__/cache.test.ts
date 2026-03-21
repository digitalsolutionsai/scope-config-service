import { ConfigCache } from '../cache';
import {
  ConfigIdentifier,
  Scope,
  ScopeConfig,
  ConfigTemplate,
  FieldType,
} from '../types';

function makeIdentifier(overrides?: Partial<ConfigIdentifier>): ConfigIdentifier {
  return {
    serviceName: 'test-service',
    scope: Scope.SYSTEM,
    groupId: 'database',
    ...overrides,
  };
}

function makeScopeConfig(overrides?: Partial<ScopeConfig>): ScopeConfig {
  return {
    versionInfo: {
      id: 1,
      identifier: makeIdentifier(),
      latestVersion: 1,
      publishedVersion: 1,
      createdAt: new Date('2024-01-01'),
      createdBy: 'admin',
      updatedAt: new Date('2024-01-01'),
      updatedBy: 'admin',
    },
    currentVersion: 1,
    fields: [
      { path: 'db.host', value: 'localhost', type: FieldType.STRING },
    ],
    ...overrides,
  };
}

function makeTemplate(overrides?: Partial<ConfigTemplate>): ConfigTemplate {
  return {
    identifier: makeIdentifier(),
    serviceLabel: 'Test Service',
    groupLabel: 'Database',
    groupDescription: 'Database configuration',
    fields: [
      {
        path: 'db.host',
        label: 'Database Host',
        description: 'The database host address',
        type: FieldType.STRING,
        defaultValue: 'localhost',
        displayOn: [Scope.SYSTEM],
        options: [],
        sortOrder: 1,
      },
    ],
    sortOrder: 1,
    ...overrides,
  };
}

describe('ConfigCache', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('constructor', () => {
    it('uses default TTL of 60000ms', () => {
      const cache = new ConfigCache();
      const id = makeIdentifier();
      const config = makeScopeConfig();

      cache.set(id, config);
      // Advance time just under default TTL
      jest.advanceTimersByTime(59999);
      const [result, isValid] = cache.get(id);
      expect(result).toEqual(config);
      expect(isValid).toBe(true);
    });

    it('accepts custom TTL', () => {
      const cache = new ConfigCache(5000);
      const id = makeIdentifier();
      const config = makeScopeConfig();

      cache.set(id, config);
      jest.advanceTimersByTime(4999);
      expect(cache.get(id)[1]).toBe(true);

      jest.advanceTimersByTime(2);
      expect(cache.get(id)[1]).toBe(false);
    });
  });

  describe('config operations', () => {
    let cache: ConfigCache;
    const id = makeIdentifier();
    const config = makeScopeConfig();

    beforeEach(() => {
      cache = new ConfigCache(10000);
    });

    it('set/get lifecycle - stores and retrieves config with isValid=true', () => {
      cache.set(id, config);
      const [result, isValid] = cache.get(id);

      expect(result).toEqual(config);
      expect(isValid).toBe(true);
    });

    it('returns [null, false] for missing config', () => {
      const [result, isValid] = cache.get(id);
      expect(result).toBeNull();
      expect(isValid).toBe(false);
    });

    it('expires entries after TTL', () => {
      cache.set(id, config);
      jest.advanceTimersByTime(10001);

      const [result, isValid] = cache.get(id);
      expect(result).toEqual(config);
      expect(isValid).toBe(false);
    });

    it('getStale returns data after expiration', () => {
      cache.set(id, config);
      jest.advanceTimersByTime(20000);

      const result = cache.getStale(id);
      expect(result).toEqual(config);
    });

    it('getStale returns null for missing entry', () => {
      expect(cache.getStale(id)).toBeNull();
    });

    it('invalidate removes a specific config', () => {
      cache.set(id, config);
      cache.invalidate(id);

      const [result, isValid] = cache.get(id);
      expect(result).toBeNull();
      expect(isValid).toBe(false);
    });
  });

  describe('template operations', () => {
    let cache: ConfigCache;
    const id = makeIdentifier();
    const template = makeTemplate();

    beforeEach(() => {
      cache = new ConfigCache(10000);
    });

    it('set/get lifecycle - stores and retrieves template with isValid=true', () => {
      cache.setTemplate(id, template);
      const [result, isValid] = cache.getTemplate(id);

      expect(result).toEqual(template);
      expect(isValid).toBe(true);
    });

    it('returns [null, false] for missing template', () => {
      const [result, isValid] = cache.getTemplate(id);
      expect(result).toBeNull();
      expect(isValid).toBe(false);
    });

    it('expires template entries after TTL', () => {
      cache.setTemplate(id, template);
      jest.advanceTimersByTime(10001);

      const [result, isValid] = cache.getTemplate(id);
      expect(result).toEqual(template);
      expect(isValid).toBe(false);
    });

    it('getTemplateStale returns data after expiration', () => {
      cache.setTemplate(id, template);
      jest.advanceTimersByTime(20000);

      const result = cache.getTemplateStale(id);
      expect(result).toEqual(template);
    });

    it('getTemplateStale returns null for missing entry', () => {
      expect(cache.getTemplateStale(id)).toBeNull();
    });

    it('invalidateTemplate removes a specific template', () => {
      cache.setTemplate(id, template);
      cache.invalidateTemplate(id);

      const [result, isValid] = cache.getTemplate(id);
      expect(result).toBeNull();
      expect(isValid).toBe(false);
    });
  });

  describe('clear', () => {
    it('clears all configs and templates', () => {
      const cache = new ConfigCache(10000);
      const id = makeIdentifier();

      cache.set(id, makeScopeConfig());
      cache.setTemplate(id, makeTemplate());

      cache.clear();

      expect(cache.get(id)[0]).toBeNull();
      expect(cache.getTemplate(id)[0]).toBeNull();
    });
  });

  describe('getCachedIdentifiers', () => {
    it('returns correct identifiers from cached configs', () => {
      const cache = new ConfigCache(10000);
      const id1 = makeIdentifier({ serviceName: 'svc-a', groupId: 'grp-1', scope: Scope.SYSTEM });
      const id2 = makeIdentifier({ serviceName: 'svc-b', groupId: 'grp-2', scope: Scope.PROJECT, projectId: 'p1' });

      cache.set(id1, makeScopeConfig());
      cache.set(id2, makeScopeConfig());

      const identifiers = cache.getCachedIdentifiers();
      expect(identifiers).toHaveLength(2);

      const svcA = identifiers.find(i => i.serviceName === 'svc-a');
      expect(svcA).toBeDefined();
      expect(svcA!.groupId).toBe('grp-1');
      expect(svcA!.scope).toBe(Scope.SYSTEM);

      const svcB = identifiers.find(i => i.serviceName === 'svc-b');
      expect(svcB).toBeDefined();
      expect(svcB!.groupId).toBe('grp-2');
      expect(svcB!.scope).toBe(Scope.PROJECT);
      expect(svcB!.projectId).toBe('p1');
    });

    it('returns empty array when no configs are cached', () => {
      const cache = new ConfigCache();
      expect(cache.getCachedIdentifiers()).toEqual([]);
    });
  });

  describe('unique cache keys', () => {
    it('different identifiers produce unique cache entries', () => {
      const cache = new ConfigCache(10000);
      const id1 = makeIdentifier({ scope: Scope.SYSTEM });
      const id2 = makeIdentifier({ scope: Scope.PROJECT, projectId: 'proj-1' });
      const config1 = makeScopeConfig({ currentVersion: 1 });
      const config2 = makeScopeConfig({ currentVersion: 2 });

      cache.set(id1, config1);
      cache.set(id2, config2);

      expect(cache.get(id1)[0]!.currentVersion).toBe(1);
      expect(cache.get(id2)[0]!.currentVersion).toBe(2);
    });
  });

  describe('background sync', () => {
    it('starts and calls sync function on interval', () => {
      const cache = new ConfigCache(60000);
      const id = makeIdentifier();
      cache.set(id, makeScopeConfig());

      const syncFn = jest.fn().mockResolvedValue(undefined);
      cache.startBackgroundSync(5000, syncFn);

      // Should not be called immediately
      expect(syncFn).not.toHaveBeenCalled();

      // Advance past one interval
      jest.advanceTimersByTime(5000);
      expect(syncFn).toHaveBeenCalledTimes(1);
      expect(syncFn).toHaveBeenCalledWith(expect.objectContaining({
        serviceName: 'test-service',
        groupId: 'database',
      }));

      cache.stopBackgroundSync();
    });

    it('stopBackgroundSync prevents further calls', () => {
      const cache = new ConfigCache(60000);
      const id = makeIdentifier();
      cache.set(id, makeScopeConfig());

      const syncFn = jest.fn().mockResolvedValue(undefined);
      cache.startBackgroundSync(5000, syncFn);

      jest.advanceTimersByTime(5000);
      expect(syncFn).toHaveBeenCalledTimes(1);

      cache.stopBackgroundSync();

      jest.advanceTimersByTime(15000);
      expect(syncFn).toHaveBeenCalledTimes(1);
    });

    it('startBackgroundSync stops previous sync before starting new one', () => {
      const cache = new ConfigCache(60000);
      const id = makeIdentifier();
      cache.set(id, makeScopeConfig());

      const syncFn1 = jest.fn().mockResolvedValue(undefined);
      const syncFn2 = jest.fn().mockResolvedValue(undefined);

      cache.startBackgroundSync(5000, syncFn1);
      cache.startBackgroundSync(5000, syncFn2);

      jest.advanceTimersByTime(5000);
      expect(syncFn1).not.toHaveBeenCalled();
      expect(syncFn2).toHaveBeenCalledTimes(1);

      cache.stopBackgroundSync();
    });

    it('stopBackgroundSync is safe to call when no sync is running', () => {
      const cache = new ConfigCache();
      expect(() => cache.stopBackgroundSync()).not.toThrow();
    });
  });
});
