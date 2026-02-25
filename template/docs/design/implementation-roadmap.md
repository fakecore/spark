# DoorX Implementation Roadmap

> **Version**: 2.0
> **Last Updated**: 2026-02-26
> **Focus**: Simplified Backend Template

## Project Status

Current version provides a complete backend template with:
- ✅ User authentication and authorization
- ✅ Role-based access control
- ✅ System management features
- ✅ File storage management
- ✅ Message system
- ✅ Audit logging
- ✅ Database migrations
- ✅ Docker deployment
- ✅ OpenAPI documentation

## Completed Features (v2.0)

### Phase 1: Core Infrastructure ✅
- [x] Kratos v2 framework setup
- [x] uber/fx dependency injection
- [x] PostgreSQL database with GORM
- [x] Redis caching
- [x] Configuration management
- [x] Logging and tracing

### Phase 2: Authentication & Authorization ✅
- [x] JWT-based authentication
- [x] Casbin RBAC system
- [x] User management
- [x] Role and permission management
- [x] Department management
- [x] OAuth provider support

### Phase 3: System Features ✅
- [x] Dictionary management
- [x] Menu management
- [x] File management with multiple storage backends
- [x] Message system
- [x] Audit logging
- [x] Operation logs

### Phase 4: Developer Experience ✅
- [x] Database migrations with Atlas
- [x] OpenAPI documentation
- [x] Docker deployment
- [x] Health check endpoints
- [x] Metrics and monitoring
- [x] Testing infrastructure

## Architecture Decisions

### Simplified Stack
We chose to simplify the architecture by:
- Removing NATS JetStream messaging (can be added later if needed)
- Removing ChangeDAG synchronization (not needed for single-instance deployments)
- Removing plugin system (can be added as microservices if needed)
- Focusing on core web application features

### Technology Choices

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Framework | Kratos v2 | Production-ready, extensible Go framework |
| DI | uber/fx | Clean dependency injection, testable code |
| Database | PostgreSQL | Reliable, feature-rich SQL database |
| ORM | GORM | Popular Go ORM with good ecosystem |
| Migrations | Atlas | Modern migration tool with automatic diffing |
| Cache | Redis | Fast in-memory caching |
| Auth | JWT + Casbin | Stateless auth with flexible RBAC |
| Storage | Multi-backend | Flexible file storage options |

## Optional Future Enhancements

These features can be added as needed:

### Messaging & Events
- Add message queue (RabbitMQ, Kafka, or NATS)
- Implement event-driven architecture
- Add async task processing

### Real-time Features
- WebSocket support
- Server-sent events
- Real-time notifications

### Advanced Features
- Multi-tenancy support
- Rate limiting
- API versioning
- GraphQL API
- Background job processing

### Integration
- Third-party service integrations
- Webhook support
- API gateway

## Development Workflow

### Adding New Features

1. **Design API**
   ```bash
   # Create/edit proto files in api/
   vim api/system/v1/new_feature.proto
   ```

2. **Generate Code**
   ```bash
   make api
   ```

3. **Implement Business Logic**
   ```bash
   # Add use case in internal/biz/
   vim internal/biz/new_feature.go
   ```

4. **Add Data Layer**
   ```bash
   # Add repository in internal/data/
   vim internal/data/new_feature_repo.go
   ```

5. **Implement Service**
   ```bash
   # Add service in internal/service/
   vim internal/service/new_feature.go
   ```

6. **Register Routes**
   ```bash
   # Edit internal/server/http.go or grpc.go
   vim internal/server/http.go
   ```

7. **Add Tests**
   ```bash
   # Add tests in respective directories
   vim internal/biz/new_feature_test.go
   ```

8. **Update Documentation**
   ```bash
   # Update README and API docs
   ```

## Database Schema Management

### Adding New Tables

1. Define GORM model
2. Run `make gendb` to generate query code
3. Create migration: `make migrate-diff NAME=add_new_table`
4. Review generated migration
5. Apply migration: `make migrate-dev`

### Modifying Tables

1. Update GORM model
2. Create migration: `make migrate-diff NAME=modify_table`
3. Review and apply migration

## Testing Strategy

### Unit Tests
- Business logic tests in `internal/biz/`
- Service layer tests in `internal/service/`
- Utility function tests in `pkg/`

### Integration Tests
- API endpoint tests in `internal/integrationtest/`
- Database tests with testcontainers
- Set `PROJECT_TEMPLATE_INTEGRATION=1` to run

## Deployment

### Development
```bash
make dev-run
```

### Production
```bash
make docker-build-prod
docker-compose up -d
```

## Monitoring

### Health Checks
- `/healthz` - Basic health check
- `/readyz` - Readiness check (dependencies)

### Metrics
- `/metrics` - Prometheus metrics endpoint
- Request/response metrics
- Database metrics
- Custom business metrics

### Logging
- Structured JSON logs
- Configurable log levels
- Request/response logging
- Audit trail

## Security Considerations

### Current Implementation
- JWT authentication
- RBAC authorization
- Password hashing
- SQL injection prevention
- XSS protection
- CSRF protection

### Recommendations for Production
- Enable HTTPS
- Set strong JWT secrets
- Configure rate limiting
- Enable audit logging
- Regular security updates
- Use secrets management
- Enable database backups
- Configure firewall rules

## Performance Optimization

### Database
- Use indexes for frequent queries
- Optimize N+1 queries
- Use prepared statements
- Configure connection pooling

### Caching
- Cache frequently accessed data
- Use Redis for distributed caching
- Configure cache TTL appropriately
- Monitor cache hit rates

### Application
- Use connection pooling
- Enable compression
- Optimize JSON serialization
- Use streaming for large responses

## Troubleshooting

### Common Issues

**Database Connection Failed**
- Check DATABASE_URL
- Verify PostgreSQL is running
- Check network connectivity

**Redis Connection Failed**
- Redis is optional (falls back to memory)
- Check Redis configuration
- Verify Redis is running

**Migration Failed**
- Check DATABASE_URL
- Verify database exists
- Review migration file
- Check for schema conflicts

## Support & Contributing

### Getting Help
- Check documentation in `docs/`
- Review example code
- Check test files for usage examples

### Contributing
1. Follow existing code patterns
2. Add tests for new features
3. Update documentation
4. Run `make check-commit` before committing
5. Ensure all tests pass

## License

[Add your license here]
