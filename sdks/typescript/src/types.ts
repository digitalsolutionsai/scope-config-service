/**
 * Example types for the ScopeConfig SDK
 * Note: In production, these should be generated from proto files
 */

import * as grpc from "@grpc/grpc-js";

// Client configuration options
export interface ClientOptions {
  address: string;
  insecure?: boolean;
  credentials?: grpc.ChannelCredentials;
  channelOptions?: grpc.ChannelOptions;
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
