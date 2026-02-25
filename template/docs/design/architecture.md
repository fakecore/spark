# DoorX Architecture

> **Version**: 2.0
> **Last Updated**: 2026-02-26
> **Status**: Simplified Backend Template

## Overview

DoorX is a Go backend template built on Kratos v2 with uber/fx dependency injection, providing a solid foundation for web applications and APIs.

## Technology Stack

### Core Framework
- **Kratos v2**: Go framework for microservices with built-in service discovery, load balancing, and tracing
- **uber/fx**: Dependency injection framework for Go applications
- **GORM**: ORM library with support for PostgreSQL, MySQL, SQLite
- **Redis**: Caching and session storage with go-redis client

### Database & Migrations
- **PostgreSQL**: Primary database (MySQL also supported)
- **Atlas**: Modern database migration tool with automatic diff generation
- **GORM Gen**: Type-safe ORM code generator

### Authentication & Authorization
- **JWT**: JSON Web Token-based authentication
- **Casbin**: Policy-based access control with RBAC support
- **RBAC**: Role-based access control with flexible permissions

### Storage & Infrastructure
- **Multi-backend OSS Support**: Local filesystem, AWS S3, Aliyun OSS, Tencent COS
- **OpenTelemetry**: Distributed tracing support
- **Prometheus**: Metrics collection and monitoring

## Architecture Layers

```
┌─────────────────────────────────────────────┐
│           API Layer (HTTP/gRPC)             │
│  ┌──────────────┐        ┌──────────────┐  │
│  │  HTTP Server │        │  gRPC Server │  │
│  └──────────────┘        └──────────────┘  │
└─────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────┐
│         Service Layer (Handlers)            │
│  ┌──────────────────────────────────────┐  │
│  │  System Services (User, Role, etc.) │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────┐
│      Business Logic Layer (Use Cases)       │
│  ┌──────────────────────────────────────┐  │
│  │  Business Logic & Domain Models      │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────┐
│         Data Layer (Repositories)           │
│  ┌─────────┐  ┌─────────┐  ┌───────────┐  │
│  │ Database│  │  Redis  │  │    OSS    │  │
│  └─────────┘  └─────────┘  └───────────┘  │
└─────────────────────────────────────────────┘
```

## Project Structure

```
project-template/
├── api/                    # API definitions (Protobuf)
│   ├── common/v1/         # Common API definitions
│   └── system/v1/         # System service APIs
├── cmd/                   # Application entry points
│   └── server/            # Main server application
├── internal/
│   ├── biz/              # Business logic layer
│   ├── data/             # Data access layer (repositories)
│   ├── server/           # HTTP/gRPC servers
│   ├── service/          # Service layer (request handlers)
│   ├── conf/             # Configuration
│   └── middleware/       # HTTP/gRPC middleware
├── pkg/                  # Shared packages
│   ├── config/           # Configuration management
│   ├── errors/           # Error handling
│   ├── metrics/          # Metrics utilities
│   └── utils/            # Utility functions
├── migrations/           # Database migrations
├── docker/              # Docker configurations
└── docs/                # Documentation
```

## Core Features

### User Management
- User CRUD operations
- Role-based access control
- Department and post management
- User authentication with JWT

### System Management
- Dictionary management
- Menu management
- File management
- Message system
- Audit logging

### Infrastructure
- Health checks (`/healthz`, `/readyz`)
- Configuration hot-reload
- Distributed tracing
- Metrics collection
- Graceful shutdown

## Configuration

Configuration is managed through YAML files with environment variable overrides:

- `docker/backend/config/config.yaml` - Main configuration
- `docker/backend/config/config.yaml.example` - Configuration template
- Environment variables with config.yaml 对应字段 prefix override YAML values

## Database Management

### Schema Management
- GORM models define the schema
- Atlas generates migrations automatically
- Type-safe queries with GORM Gen

### Migration Workflow
```bash
# Generate migration from schema changes
make migrate-diff NAME=add_user_phone

# Apply migrations
make migrate-dev  # Development
make migrate-prod # Production

# Check migration status
make migrate-status
```

## API Design

### REST API
- OpenAPI 3.0 specification
- Auto-generated from Protobuf definitions
- Validation via `protoc-gen-validate`

### gRPC API
- Protobuf service definitions
- Streaming support
- Inter-service communication

## Deployment

### Local Development
```bash
make infra-up      # Start PostgreSQL & Redis
make migrate-dev   # Run migrations
make run          # Start server
```

### Docker Deployment
```bash
make docker-build-prod
docker-compose up -d
```

## Security

### Authentication
- JWT-based stateless authentication
- Access token expiration
- Client token support
- Multiple OAuth providers

### Authorization
- Casbin RBAC model
- Role-permission mapping
- Department-based access control
- API-level permission checks

### Data Security
- Password hashing with bcrypt
- SQL injection prevention (GORM)
- XSS protection
- CSRF protection

## Monitoring & Observability

### Metrics
- Prometheus endpoint `/metrics`
- Request/response metrics
- Database connection pool metrics
- Custom business metrics

### Tracing
- OpenTelemetry integration
- Distributed tracing context propagation
- Request timing analysis

### Logging
- Structured logging with zap
- Log levels (debug, info, warn, error)
- Request/response logging
- Audit logging

## Testing

### Unit Tests
```bash
make test  # Run all tests
```

### Integration Tests
```bash
PROJECT_TEMPLATE_INTEGRATION=1 go test ./internal/integrationtest/...
```

## Performance

### Caching Strategy
- Redis for distributed caching
- In-memory fallback when Redis unavailable
- Configurable cache modes (external, auto, memory)

### Database Optimization
- Connection pooling
- Query optimization with indexes
- Prepared statements
- Batch operations

## Future Extensions

The architecture supports easy addition of:
- Additional storage backends
- New authentication providers
- Custom middleware
- Additional services
- Event-driven components
- Message queue integration

## Contributing

When adding new features:
1. Define APIs in `api/` directory
2. Implement business logic in `internal/biz/`
3. Add data layer in `internal/data/`
4. Create service handlers in `internal/service/`
5. Register routes in `internal/server/`
6. Add tests
7. Update documentation

## License

[Add your license here]
