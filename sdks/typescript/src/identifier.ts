import { ConfigIdentifier, Scope } from './types';

/**
 Fluent builder for ConfigIdentifier
 
  @example
  ```typescript
  const identifier = new IdentifierBuilder('my-service')
    .withScope(Scope.SYSTEM)
    .withGroupId('database')
    .build();
  ```
 */
export class IdentifierBuilder {
  private identifier: ConfigIdentifier;

  constructor(serviceName: string) {
    this.identifier = {
      serviceName,
      scope: Scope.SCOPE_UNSPECIFIED,
      groupId: '',
    };
  }

  withScope(scope: Scope): this {
    this.identifier.scope = scope;
    return this;
  }

  withGroupId(groupId: string): this {
    this.identifier.groupId = groupId;
    return this;
  }

  withProjectId(projectId: string): this {
    this.identifier.projectId = projectId;
    return this;
  }

  withStoreId(storeId: string): this {
    this.identifier.storeId = storeId;
    return this;
  }

  withUserId(userId: string): this {
    this.identifier.userId = userId;
    return this;
  }

  build(): ConfigIdentifier {
    return { ...this.identifier };
  }
}

export function createIdentifier(serviceName: string): IdentifierBuilder {
  return new IdentifierBuilder(serviceName);
}
