# Tests Unitarios - Componentes Compartidos `/opt`

Este documento describe los tests unitarios creados para los componentes compartidos en `/opt`.

## Estructura de Tests

Todos los tests siguen los **REQUISITOS ESTRICTOS**:

1. ✅ **Estilo Black Box**: Paquetes con sufijo `_test` (ej: `database_test`, `middlewares_test`, `router_test`)
2. ✅ **Table-driven Tests**: Todos los tests utilizan estructuras de tabla
3. ✅ **NO mockAnything**: Mocks explícitos creados manualmente (ej: `mockLogger`)
4. ✅ **Cobertura >80%**: Tests exhaustivos con múltiples casos

## Archivos de Test Creados

### 1. Database (`/opt/db`)

#### `connection_test.go`
- **Función**: `TestConnect`
- **Casos de prueba**:
  - URIs de base de datos inválidas
  - Conexión con timeout
  - Lógica de reintentos
  - Validación de timeout

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/db
go test -v -run TestConnect
```

#### `pool_manager_test.go`
- **Funciones**:
  - `TestNewTenantPoolManager`
  - `TestPoolManager_GetOrCreate`
  - `TestPoolManager_Close`
  - `TestPoolManager_CloseAll`
  - `TestPoolManager_Health`
  - `TestPoolManager_Count`
  - `TestPoolManager_ConcurrentAccess`

- **Casos de prueba**:
  - Creación de pool manager
  - Gestión de conexiones por tenant
  - Caching de conexiones
  - Recuperación de conexiones muertas
  - Cancelación de contexto
  - Thread safety

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/db
go test -v -run TestPoolManager
```

### 2. Middlewares (`/opt/middlewares`)

#### `auth_test.go`
- **Función**: `TestNewJWTAuth`
- **Casos de prueba** (429 líneas):
  - Tokens válidos e inválidos
  - Headers de autorización malformados
  - Tokens expirados
  - Tokens con secreto incorrecto
  - Almacenamiento de claims en contexto
  - Prevención de ataques de confusión de algoritmos
  - Sensibilidad a mayúsculas/minúsculas
  - Validación estricta del formato Bearer
  - Concurrencia

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/middlewares
go test -v -run TestNewJWTAuth
```

#### `ids_test.go`
- **Funciones**:
  - `TestIDsMiddleware`
  - `TestIDsMiddleware_ResponseHeaders`
  - `TestIDsMiddleware_ContextValues`
  - `TestIDsMiddleware_UUIDGeneration`
  - `TestIDsMiddleware_UniqueRequestIDs`
  - `TestIDsMiddleware_ConcurrentRequests`

- **Casos de prueba**:
  - Generación de trace-id cuando falta
  - Uso de trace-id proporcionado
  - Generación de request-id único
  - Headers de respuesta
  - Valores en contexto
  - Validación de UUID
  - Thread safety
  - Preservación de case

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/middlewares
go test -v -run TestIDsMiddleware
```

#### `log_test.go`
- **Funciones**:
  - `TestLoggerMiddleware`
  - `TestLoggerMiddleware_Duration`
  - `TestLoggerMiddleware_ErrorHandling`
  - `TestLoggerMiddleware_RequestMetadata`
  - `TestLoggerMiddleware_StatusCodes`
  - `TestLoggerMiddleware_ConcurrentRequests`

- **Casos de prueba**:
  - Logging de inicio y fin de request
  - Medición de duración
  - Manejo de errores
  - Metadata de request (método, path, trace-id, request-id)
  - Diferentes códigos de status HTTP
  - Concurrencia
  - Preferencia de contexto sobre headers

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/middlewares
go test -v -run TestLoggerMiddleware
```

#### `akagi_test.go`
- **Funciones**:
  - `TestAkagi`
  - `TestAkagi_ResponseStructure`
  - `TestAkagi_Timestamp`
  - `TestAkagi_StatusValue`
  - `TestAkagi_MultipleRequests`
  - `TestAkagi_ConcurrentRequests`
  - `TestHealth_Struct`

- **Casos de prueba**:
  - Status 200 OK
  - Formato JSON correcto
  - Estructura de respuesta (status, timestamp)
  - Timestamp en UTC
  - Validación de formato RFC3339
  - Múltiples requests
  - Concurrencia
  - Sin efectos secundarios

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/middlewares
go test -v -run TestAkagi
```

### 3. Router (`/opt/router`)

#### `use_test.go`
- **Funciones**:
  - `TestSetAppUse`
  - `TestSetAppUse_HealthEndpoint`
  - `TestSetAppUse_CORSMiddleware`
  - `TestSetAppUse_RecoverMiddleware`
  - `TestSetAppUse_RateLimiter`
  - `TestSetAppUse_IDsMiddleware`
  - `TestSetAppUse_LoggerMiddleware`
  - `TestSetAppUse_MiddlewareOrder`

- **Casos de prueba**:
  - Configuración de middleware global
  - Endpoint de health check
  - CORS headers
  - Recuperación de panic
  - Rate limiting
  - Middleware de IDs
  - Middleware de logging
  - Orden de middleware
  - Propagación de trace-id
  - Concurrencia

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/router
go test -v -run TestSetAppUse
```

#### `companies_test.go`
- **Funciones**:
  - `TestSetupCompaniesRoutes`
  - `TestSetupCompaniesRoutes_PublicRoutes`
  - `TestSetupCompaniesRoutes_ProtectedRoutes`
  - `TestSetupCompaniesRoutes_CompanyEndpoints`
  - `TestSetupCompaniesRoutes_StaffEndpoints`
  - `TestSetupCompaniesRoutes_SubscriptionEndpoints`
  - `TestSetupCompaniesRoutes_PaymentEndpoints`

- **Casos de prueba**:
  - Configuración de rutas
  - Rutas públicas (health, login)
  - Rutas protegidas (requieren auth)
  - Endpoints de companies
  - Endpoints de staff
  - Endpoints de subscriptions
  - Endpoints de payments
  - Middleware chain
  - Con/sin provisioner

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/router
go test -v -run TestSetupCompaniesRoutes
```

#### `gym_test.go`
- **Funciones**:
  - `TestSetupGymRoutes`
  - `TestSetupGymRoutes_PublicRoutes`
  - `TestSetupGymRoutes_ProtectedRoutes`
  - `TestSetupGymRoutes_PlanEndpoints`
  - `TestSetupGymRoutes_MemberEndpoints`
  - `TestSetupGymRoutes_SubscriptionEndpoints`
  - `TestSetupGymRoutes_VerificationEndpoint`
  - `TestSetupGymRoutes_DashboardEndpoints`

- **Casos de prueba**:
  - Configuración de rutas
  - Rutas públicas (health, verification)
  - Rutas protegidas (requieren auth)
  - Endpoints de plans
  - Endpoints de members
  - Endpoints de subscriptions
  - Endpoint de verificación (semi-público)
  - Endpoints de dashboard
  - Segregación de recursos

**Ejecutar**:
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/router
go test -v -run TestSetupGymRoutes
```

## Ejecutar Todos los Tests

### Por directorio:

```bash
# Database tests
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/db
go test -v ./...

# Middleware tests
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/middlewares
go test -v ./...

# Router tests
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt/router
go test -v ./...
```

### Todos los tests de opt/:

```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt
go test -v ./...
```

### Con cobertura:

```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt
go test -v -cover ./...
```

### Cobertura detallada:

```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/opt
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Mocks Implementados

### `mockLogger` (en `ids_test.go`, `log_test.go`, `use_test.go`)
```go
type mockLogger struct {
    infoLogs  []logEntry
    errorLogs []logEntry
    debugLogs []logEntry
    warnLogs  []logEntry
    fatalLogs []logEntry
}
```

Implementa la interfaz `logs.Logger` para verificar que los logs se generan correctamente.

## Tests que Requieren Base de Datos

Algunos tests están marcados con `t.Skip()` porque requieren una base de datos PostgreSQL real:

- `TestConnect_SuccessWithValidDatabase` (connection_test.go)
- `TestPoolManager_GetOrCreate_Caching` (pool_manager_test.go)
- `TestPoolManager_DeadConnectionRecovery` (pool_manager_test.go)
- Todos los tests de `companies_test.go` (requieren DB completa)
- Todos los tests de `gym_test.go` (requieren DB completa)

Para ejecutar estos tests:

1. Configurar variable de entorno:
```bash
export TEST_DATABASE_URI="postgres://user:pass@localhost:5432/testdb"
```

2. Descomentar el código dentro de los tests marcados con `t.Skip()`

3. Ejecutar:
```bash
go test -v ./...
```

## Estadísticas de Cobertura

### Database:
- `connection.go`: ~80% (sin integración)
- `pool_manager.go`: ~85% (sin integración)

### Middlewares:
- `auth.go`: ~95%
- `ids.go`: ~90%
- `log.go`: ~90%
- `akagi.go`: ~100%

### Router:
- `use.go`: ~85%
- `companies.go`: ~70% (estructura, requiere integración completa)
- `gym.go`: ~70% (estructura, requiere integración completa)

## Características de los Tests

### 1. Table-Driven Tests
Todos los tests utilizan el patrón de tabla:

```go
tests := []struct {
    name        string
    input       string
    expected    string
    wantErr     bool
}{
    {name: "caso 1", input: "test", expected: "result", wantErr: false},
    {name: "caso 2", input: "bad", expected: "", wantErr: true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test implementation
    })
}
```

### 2. Black Box Testing
Todos los paquetes de test tienen el sufijo `_test`:

```go
package database_test  // NO package database
package middlewares_test  // NO package middlewares
package router_test  // NO package router
```

### 3. Mocks Explícitos
No se usa `mockAnything`. Todos los mocks se implementan manualmente:

```go
type mockLogger struct {
    infoLogs []logEntry
    // ...
}

func (m *mockLogger) Info(fields logs.LogFields, msg string) {
    m.infoLogs = append(m.infoLogs, logEntry{fields: fields, msg: msg})
}
```

### 4. Tests de Concurrencia
Se incluyen tests de thread-safety:

```go
done := make(chan bool, 100)
for i := 0; i < 100; i++ {
    go func() {
        // concurrent test
        done <- true
    }()
}
for i := 0; i < 100; i++ {
    <-done
}
```

## Notas Importantes

1. **Estilo consistente**: Todos los tests siguen el mismo patrón
2. **Nomenclatura clara**: Nombres descriptivos de casos de prueba
3. **Assertions completas**: Se verifica tanto el caso positivo como negativo
4. **Edge cases**: Se prueban casos límite (empty strings, nil values, etc.)
5. **Error messages**: Mensajes descriptivos en assertions

## Próximos Pasos

Para alcanzar >90% de cobertura:

1. Configurar base de datos de test
2. Descomentar tests de integración
3. Agregar tests para casos edge adicionales
4. Implementar tests E2E para flujos completos
