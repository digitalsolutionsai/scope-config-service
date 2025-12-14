// Package httpgateway provides HTTP REST API endpoints for the Scope Configuration Service.
//
// @title Scope Configuration Service API
// @version 1.0
// @description HTTP REST API for managing configuration templates and values across different scopes (SYSTEM, PROJECT, STORE, USER).
// @description
// @description This service provides a REST API wrapper around the gRPC Scope Configuration Service.
// @description It enables frontend applications and external clients to interact with configurations using simple HTTP/JSON requests.
// @description
// @description ## Architecture
// @description ```
// @description Client → API Gateway (Auth/RBAC) → Config Service (Public APIs)
// @description ```
// @description
// @description ## Authentication
// @description Authentication is handled at the API Gateway level (e.g., Spring Cloud Gateway, Kong, Nginx).
// @description This service exposes public APIs and relies on an upstream gateway for authentication and authorization.
//
// @contact.name API Support
// @contact.email support@dsai.vn
//
// @host localhost:8080
// @BasePath /api/v1
//
// @schemes http https
//
// @x-extension-openapi {"example": "value on a json format"}

package httpgateway
