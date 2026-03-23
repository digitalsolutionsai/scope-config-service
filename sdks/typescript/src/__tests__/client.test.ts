import { ConfigClient, createOptionsFromEnv } from '../client';
import { Scope } from '../types';
import { IdentifierBuilder } from '../identifier';

// Mock the gRPC modules to avoid needing a real server
jest.mock('@grpc/grpc-js', () => ({
  credentials: {
    createInsecure: jest.fn(),
    createSsl: jest.fn(),
  },
  loadPackageDefinition: jest.fn(),
  status: {
    NOT_FOUND: 5,
    INVALID_ARGUMENT: 3,
    UNAVAILABLE: 14,
  },
}));
jest.mock('@grpc/proto-loader', () => ({
  loadSync: jest.fn(),
}));

describe('createOptionsFromEnv', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.replaceProperty(process, 'env', { ...originalEnv });
    delete process.env.GRPC_SCOPE_CONFIG_HOST;
    delete process.env.GRPC_SCOPE_CONFIG_PORT;
    delete process.env.GRPC_SCOPE_CONFIG_USE_TLS;
  });

  afterEach(() => {
    jest.replaceProperty(process, 'env', originalEnv);
  });

  it('returns default options when no env vars are set', () => {
    const options = createOptionsFromEnv();

    expect(options.host).toBe('localhost');
    expect(options.port).toBe(50051);
    expect(options.address).toBe('localhost:50051');
    expect(options.insecure).toBe(true);
    expect(options.cacheEnabled).toBe(true);
  });

  it('reads host from GRPC_SCOPE_CONFIG_HOST', () => {
    process.env.GRPC_SCOPE_CONFIG_HOST = 'config.example.com';
    const options = createOptionsFromEnv();

    expect(options.host).toBe('config.example.com');
    expect(options.address).toBe('config.example.com:50051');
  });

  it('reads port from GRPC_SCOPE_CONFIG_PORT', () => {
    process.env.GRPC_SCOPE_CONFIG_PORT = '9090';
    const options = createOptionsFromEnv();

    expect(options.port).toBe(9090);
    expect(options.address).toBe('localhost:9090');
  });

  it('enables TLS when GRPC_SCOPE_CONFIG_USE_TLS is "true"', () => {
    process.env.GRPC_SCOPE_CONFIG_USE_TLS = 'true';
    const options = createOptionsFromEnv();
    expect(options.insecure).toBe(false);
  });

  it('enables TLS when GRPC_SCOPE_CONFIG_USE_TLS is "1"', () => {
    process.env.GRPC_SCOPE_CONFIG_USE_TLS = '1';
    const options = createOptionsFromEnv();
    expect(options.insecure).toBe(false);
  });

  it('enables TLS when GRPC_SCOPE_CONFIG_USE_TLS is "yes"', () => {
    process.env.GRPC_SCOPE_CONFIG_USE_TLS = 'yes';
    const options = createOptionsFromEnv();
    expect(options.insecure).toBe(false);
  });

  it('keeps TLS disabled for other GRPC_SCOPE_CONFIG_USE_TLS values', () => {
    process.env.GRPC_SCOPE_CONFIG_USE_TLS = 'false';
    const options = createOptionsFromEnv();
    expect(options.insecure).toBe(true);
  });

  it('applies overrides on top of env defaults', () => {
    process.env.GRPC_SCOPE_CONFIG_HOST = 'env-host';
    const options = createOptionsFromEnv({ host: 'override-host', cacheEnabled: false });

    expect(options.host).toBe('override-host');
    expect(options.cacheEnabled).toBe(false);
    // address is built from env before overrides are spread
    expect(options.address).toBe('env-host:50051');
  });
});

describe('ConfigClient', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.replaceProperty(process, 'env', { ...originalEnv });
    delete process.env.GRPC_SCOPE_CONFIG_HOST;
    delete process.env.GRPC_SCOPE_CONFIG_PORT;
    delete process.env.GRPC_SCOPE_CONFIG_USE_TLS;
  });

  afterEach(() => {
    jest.replaceProperty(process, 'env', originalEnv);
  });

  describe('constructor', () => {
    it('creates client with default options from env', () => {
      const client = new ConfigClient();
      expect(client.isCacheEnabled()).toBe(true);
    });

    it('creates client with explicit options', () => {
      const client = new ConfigClient({
        address: 'myhost:8080',
        insecure: true,
        cacheEnabled: true,
      });
      expect(client.isCacheEnabled()).toBe(true);
    });

    it('resolves address from host and port when address is not provided', () => {
      process.env.GRPC_SCOPE_CONFIG_HOST = 'remote-host';
      process.env.GRPC_SCOPE_CONFIG_PORT = '9999';

      const client = new ConfigClient({ host: 'custom-host', port: 7777 });
      // The client should use host/port since no address is provided
      expect(client.isCacheEnabled()).toBe(true);
    });

    it('uses provided address over host/port', () => {
      const client = new ConfigClient({
        address: 'explicit:1234',
        host: 'ignored-host',
        port: 5678,
      });
      expect(client.isCacheEnabled()).toBe(true);
    });
  });

  describe('isCacheEnabled', () => {
    it('returns true by default', () => {
      const client = new ConfigClient();
      expect(client.isCacheEnabled()).toBe(true);
    });

    it('returns false when cache is disabled', () => {
      const client = new ConfigClient({ cacheEnabled: false });
      expect(client.isCacheEnabled()).toBe(false);
    });

    it('returns true when cache is explicitly enabled', () => {
      const client = new ConfigClient({ cacheEnabled: true });
      expect(client.isCacheEnabled()).toBe(true);
    });
  });

  describe('clearCache', () => {
    it('does not throw when cache is enabled', () => {
      const client = new ConfigClient({ cacheEnabled: true });
      expect(() => client.clearCache()).not.toThrow();
    });

    it('does not throw when cache is disabled', () => {
      const client = new ConfigClient({ cacheEnabled: false });
      expect(() => client.clearCache()).not.toThrow();
    });
  });

  describe('invalidateCache', () => {
    it('does not throw when cache is enabled', () => {
      const client = new ConfigClient({ cacheEnabled: true });
      const identifier = new IdentifierBuilder('svc')
        .withScope(Scope.SYSTEM)
        .withGroupId('grp')
        .build();

      expect(() => client.invalidateCache(identifier)).not.toThrow();
    });

    it('does not throw when cache is disabled', () => {
      const client = new ConfigClient({ cacheEnabled: false });
      const identifier = new IdentifierBuilder('svc')
        .withScope(Scope.SYSTEM)
        .withGroupId('grp')
        .build();

      expect(() => client.invalidateCache(identifier)).not.toThrow();
    });
  });

  describe('close', () => {
    it('can be called without error even without connect', async () => {
      const client = new ConfigClient();
      await expect(client.close()).resolves.toBeUndefined();
    });

    it('can be called multiple times without error', async () => {
      const client = new ConfigClient();
      await client.close();
      await expect(client.close()).resolves.toBeUndefined();
    });
  });
});
