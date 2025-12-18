/**
 * Types for the ScopeConfig SDK
 * Note: In production, these should be generated from proto files
 */

import * as grpc from "@grpc/grpc-js";

// Client configuration options
export interface ClientOptions {
  address: string;
  insecure?: boolean;
  credentials?: grpc.ChannelCredentials;
  channelOptions?: grpc.ChannelOptions;
  /** Enable in-memory caching (default: false) */
  cacheEnabled?: boolean;
  /** Cache TTL in milliseconds (default: 60000 = 1 minute) */
  cacheTtlMs?: number;
  /** Enable background sync (default: false) */
  backgroundSyncEnabled?: boolean;
  /** Background sync interval in milliseconds (default: 30000 = 30 seconds) */
  backgroundSyncIntervalMs?: number;
}

// Scope levels for configuration
export enum Scope {
  SCOPE_UNSPECIFIED = 0,
  SYSTEM = 1,
  PROJECT = 2,
  STORE = 3,
  USER = 4,
}

// Configuration identifier
export interface ConfigIdentifier {
  serviceName: string;
  scope: Scope;
  groupId: string;
  projectId?: string;
  storeId?: string;
  userId?: string;
}

// Configuration field types
export enum FieldType {
  FIELD_TYPE_UNSPECIFIED = 0,
  STRING = 1,
  INT = 2,
  FLOAT = 3,
  BOOLEAN = 4,
  JSON = 5,
  ARRAY_STRING = 6,
  SECRET = 7,
}

// Configuration field
export interface ConfigField {
  path: string;
  value: string;
  type: FieldType;
}

// Configuration version info
export interface ConfigVersion {
  id: number;
  identifier: ConfigIdentifier;
  latestVersion: number;
  publishedVersion: number;
  createdAt: Date;
  createdBy: string;
  updatedAt: Date;
  updatedBy: string;
}

// Complete configuration with fields
export interface ScopeConfig {
  versionInfo: ConfigVersion;
  currentVersion: number;
  fields: ConfigField[];
}

// Value option for template fields
export interface ValueOption {
  value: string;
  label: string;
}

// Template field definition
export interface ConfigFieldTemplate {
  path: string;
  label: string;
  description: string;
  type: FieldType;
  defaultValue: string;
  displayOn: Scope[];
  options: ValueOption[];
  sortOrder: number;
}

// Configuration template
export interface ConfigTemplate {
  identifier: ConfigIdentifier;
  serviceLabel: string;
  groupLabel: string;
  groupDescription: string;
  fields: ConfigFieldTemplate[];
  sortOrder: number;
}

// Options for GetValue method
export interface GetValueOptions {
  /** Use default value from template if config value is not set */
  useDefault?: boolean;
  /** Traverse parent scopes to find the value (USER -> STORE -> PROJECT -> SYSTEM) */
  inherit?: boolean;
}

// Error wrapper for gRPC errors
export class ConfigServiceError extends Error {
  constructor(
    message: string,
    public readonly code: grpc.status,
    public readonly details?: string
  ) {
    super(message);
    this.name = "ConfigServiceError";
  }
}
