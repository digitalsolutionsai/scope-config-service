import { IdentifierBuilder, createIdentifier } from '../identifier';
import { Scope } from '../types';

describe('IdentifierBuilder', () => {
  describe('constructor', () => {
    it('creates identifier with correct default values', () => {
      const builder = new IdentifierBuilder('my-service');
      const identifier = builder.build();

      expect(identifier.serviceName).toBe('my-service');
      expect(identifier.scope).toBe(Scope.SCOPE_UNSPECIFIED);
      expect(identifier.groupId).toBe('');
      expect(identifier.projectId).toBeUndefined();
      expect(identifier.storeId).toBeUndefined();
      expect(identifier.userId).toBeUndefined();
    });

    it('handles empty service name', () => {
      const builder = new IdentifierBuilder('');
      const identifier = builder.build();

      expect(identifier.serviceName).toBe('');
      expect(identifier.scope).toBe(Scope.SCOPE_UNSPECIFIED);
      expect(identifier.groupId).toBe('');
    });
  });

  describe('withScope', () => {
    it('sets scope and returns this for chaining', () => {
      const builder = new IdentifierBuilder('svc');
      const result = builder.withScope(Scope.SYSTEM);

      expect(result).toBe(builder);
      expect(result.build().scope).toBe(Scope.SYSTEM);
    });

    it.each([
      ['SYSTEM', Scope.SYSTEM],
      ['PROJECT', Scope.PROJECT],
      ['STORE', Scope.STORE],
      ['USER', Scope.USER],
      ['SCOPE_UNSPECIFIED', Scope.SCOPE_UNSPECIFIED],
    ])('sets %s scope correctly', (_name, scope) => {
      const identifier = new IdentifierBuilder('svc').withScope(scope).build();
      expect(identifier.scope).toBe(scope);
    });
  });

  describe('withGroupId', () => {
    it('sets groupId and returns this for chaining', () => {
      const builder = new IdentifierBuilder('svc');
      const result = builder.withGroupId('database');

      expect(result).toBe(builder);
      expect(result.build().groupId).toBe('database');
    });
  });

  describe('withProjectId', () => {
    it('sets projectId and returns this for chaining', () => {
      const builder = new IdentifierBuilder('svc');
      const result = builder.withProjectId('proj-123');

      expect(result).toBe(builder);
      expect(result.build().projectId).toBe('proj-123');
    });
  });

  describe('withStoreId', () => {
    it('sets storeId and returns this for chaining', () => {
      const builder = new IdentifierBuilder('svc');
      const result = builder.withStoreId('store-456');

      expect(result).toBe(builder);
      expect(result.build().storeId).toBe('store-456');
    });
  });

  describe('withUserId', () => {
    it('sets userId and returns this for chaining', () => {
      const builder = new IdentifierBuilder('svc');
      const result = builder.withUserId('user-789');

      expect(result).toBe(builder);
      expect(result.build().userId).toBe('user-789');
    });
  });

  describe('build', () => {
    it('returns a copy, not a reference to internal state', () => {
      const builder = new IdentifierBuilder('svc').withGroupId('grp');
      const first = builder.build();
      const second = builder.build();

      expect(first).toEqual(second);
      expect(first).not.toBe(second);
    });

    it('multiple builds produce independent copies', () => {
      const builder = new IdentifierBuilder('svc');
      const first = builder.withScope(Scope.SYSTEM).build();

      builder.withScope(Scope.PROJECT);
      const second = builder.build();

      expect(first.scope).toBe(Scope.SYSTEM);
      expect(second.scope).toBe(Scope.PROJECT);
    });
  });

  describe('full chaining', () => {
    it('chains all methods for SYSTEM scope', () => {
      const identifier = new IdentifierBuilder('payment-service')
        .withScope(Scope.SYSTEM)
        .withGroupId('database')
        .build();

      expect(identifier).toEqual({
        serviceName: 'payment-service',
        scope: Scope.SYSTEM,
        groupId: 'database',
      });
    });

    it('chains all methods for PROJECT scope', () => {
      const identifier = new IdentifierBuilder('payment-service')
        .withScope(Scope.PROJECT)
        .withGroupId('database')
        .withProjectId('proj-1')
        .build();

      expect(identifier).toEqual({
        serviceName: 'payment-service',
        scope: Scope.PROJECT,
        groupId: 'database',
        projectId: 'proj-1',
      });
    });

    it('chains all methods for STORE scope', () => {
      const identifier = new IdentifierBuilder('payment-service')
        .withScope(Scope.STORE)
        .withGroupId('database')
        .withProjectId('proj-1')
        .withStoreId('store-1')
        .build();

      expect(identifier).toEqual({
        serviceName: 'payment-service',
        scope: Scope.STORE,
        groupId: 'database',
        projectId: 'proj-1',
        storeId: 'store-1',
      });
    });

    it('chains all methods for USER scope with all fields', () => {
      const identifier = new IdentifierBuilder('payment-service')
        .withScope(Scope.USER)
        .withGroupId('preferences')
        .withProjectId('proj-1')
        .withStoreId('store-1')
        .withUserId('user-42')
        .build();

      expect(identifier).toEqual({
        serviceName: 'payment-service',
        scope: Scope.USER,
        groupId: 'preferences',
        projectId: 'proj-1',
        storeId: 'store-1',
        userId: 'user-42',
      });
    });
  });
});

describe('createIdentifier', () => {
  it('returns an IdentifierBuilder instance', () => {
    const builder = createIdentifier('my-service');
    expect(builder).toBeInstanceOf(IdentifierBuilder);
  });

  it('sets the service name correctly', () => {
    const identifier = createIdentifier('my-service').build();
    expect(identifier.serviceName).toBe('my-service');
  });

  it('supports full chaining', () => {
    const identifier = createIdentifier('svc')
      .withScope(Scope.PROJECT)
      .withGroupId('cache')
      .withProjectId('p1')
      .build();

    expect(identifier.serviceName).toBe('svc');
    expect(identifier.scope).toBe(Scope.PROJECT);
    expect(identifier.groupId).toBe('cache');
    expect(identifier.projectId).toBe('p1');
  });
});
