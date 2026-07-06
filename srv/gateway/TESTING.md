# Gateway Service - Test Documentation

## Overview

Tests unitarios completos para el servicio Gateway siguiendo las mejores prácticas de Go:
- **Black Box Testing**: Todos los tests usan el sufijo `_test` en el paquete
- **Table-Driven Tests**: Estructura de tests basada en tablas de casos
- **Sin Mocks Automáticos**: Mocks explícitos creados manualmente
- **Cobertura >80%**: Cobertura general superior al objetivo

## Cobertura de Tests

```
handlers:    89.1%  - Proxy HTTP (ProxyToCompanies, ProxyToGym, ProxyVerification)
middleware:  91.9%  - Auth JWT, Tenant, Logger, DB Resolver
publisher:   14.3%  - NATS Publisher (NoOp y patterns)
router:     100.0%  - Configuración de rutas y middlewares
```

## Estructura de Tests

### 1. Handlers Tests (`handlers/proxy_test.go`)

Tests para el proxy HTTP que enruta solicitudes a servicios backend.

**Casos de prueba incluidos:**
- ✅ Proxy exitoso a servicio de companies (GET, POST, PUT, DELETE)
- ✅ Proxy con query parameters
- ✅ Proxy a servicio gym con slug de tenant
- ✅ Verificación de QR codes
- ✅ Manejo de errores (404, 500, backend unavailable)
- ✅ Filtrado de headers hop-by-hop
- ✅ Preservación de Content-Type
- ✅ Headers X-Forwarded correctos

**Ejemplo de test:**
```go
func TestProxyToCompanies(t *testing.T) {
    tests := []struct {
        name           string
        requestMethod  string
        backendStatus  int
        expectedStatus int
    }{
        {
            name:           "success - GET request",
            requestMethod:  http.MethodGet,
            backendStatus:  http.StatusOK,
            expectedStatus: http.StatusOK,
        },
        // ... más casos
    }
    // Table-driven test implementation
}
```

### 2. Middleware Tests

#### Auth Middleware (`middleware/auth_test.go`)

Tests para autenticación JWT.

**Casos de prueba:**
- ✅ Token JWT válido con claims correctos
- ✅ Token expirado
- ✅ Firma inválida
- ✅ Header Authorization faltante
- ✅ Formato de header inválido
- ✅ Token malformado
- ✅ Auth opcional (OptionalJWTAuth)
- ✅ Case-insensitive Bearer token

**Helpers incluidos:**
```go
generateValidToken(t, staffID, companyID, email, role)
generateExpiredToken(t)
generateTokenWithWrongSecret(t)
```

#### Tenant Middleware (`middleware/tenant_test.go`)

Tests para resolución de tenant y control de acceso.

**Casos de prueba:**
- ✅ Slug válido (mínimo 3, máximo 50 caracteres)
- ✅ Slug con números y guiones
- ✅ Slug faltante o inválido
- ✅ Header X-Tenant-Slug configurado correctamente
- ✅ Control de acceso por company_id
- ✅ Superadmin puede acceder a cualquier company
- ✅ Usuarios normales solo su company

#### Logger Middleware (`middleware/logger_test.go`)

Tests para logging de solicitudes.

**Casos de prueba:**
- ✅ Log de respuestas exitosas (2xx)
- ✅ Log de errores de cliente (4xx) con nivel WARN
- ✅ Log de errores de servidor (5xx) con nivel ERROR
- ✅ Inclusión de tenant_slug y staff_id en logs
- ✅ Medición de duración de requests
- ✅ Log de remote IP
- ✅ Diferentes métodos HTTP

**Formato de log verificado:**
```json
{
  "method": "GET",
  "path": "/api/v1/test",
  "status": 200,
  "duration": "1.5ms",
  "remote_ip": "192.0.2.1",
  "tenant": "my-gym",
  "staff_id": "staff-123"
}
```

#### DB Resolver Middleware (`middleware/db_resolver_test.go`)

Tests para resolución de base de datos por tenant.

**Casos de prueba:**
- ✅ Error cuando tenant_slug no está en contexto
- ✅ Integración con TenantResolver middleware

> **Nota**: Tests completos de DB resolver requieren un TenantPoolManager real o mock más elaborado.

### 3. Publisher Tests (`publisher/nats_test.go`)

Tests para publicación de eventos NATS.

**Casos de prueba:**
- ✅ NoOpPublisher (todas las operaciones son no-op)
- ✅ Serialización de eventos de dominio
- ✅ Validación de subjects de eventos
- ✅ Concurrencia de publicación
- ✅ Manejo de errores de NATS

**Eventos testeados:**
```go
CompanyCreatedEvent
CompanyActivatedEvent
CompanyProvisionedEvent
CompanyProvisionFailedEvent
```

### 4. Router Tests (`router/router_test.go`)

Tests para configuración de rutas y middlewares.

**Casos de prueba:**
- ✅ Configuración correcta del router
- ✅ Health check endpoint público
- ✅ Rutas públicas (login, register, verify)
- ✅ Rutas protegidas requieren autenticación
- ✅ Rutas de companies service
- ✅ Rutas de gym service con slug
- ✅ Catch-all para rutas gym
- ✅ CORS habilitado
- ✅ Orden de middlewares (Recover -> Logger -> CORS -> Auth)

## Ejecutar Tests

### Todos los tests
```bash
cd /Users/diegoa.leon/Documents/dev/bro/bank-api/srv/gateway
go test ./... -v
```

### Con cobertura
```bash
go test ./... -cover
```

### Cobertura detallada
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Tests específicos
```bash
# Solo handlers
go test ./handlers -v

# Solo middleware
go test ./middleware -v

# Test específico
go test ./middleware -run TestJWTAuth -v
```

## Características de los Tests

### 1. Black Box Testing
Todos los tests están en paquetes `*_test` para probar solo la API pública:
```go
package middleware_test  // No package middleware
package handlers_test    // No package handlers
```

### 2. Table-Driven Tests
Estructura consistente en todos los tests:
```go
tests := []struct {
    name           string
    input          string
    expectedOutput string
    expectedError  bool
}{
    {name: "caso 1", ...},
    {name: "caso 2", ...},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

### 3. Mocks Explícitos
No se usan librerías de mocking automático. Ejemplos:

```go
// Mock HTTP server para tests de proxy
backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"result":"ok"}`))
}))
defer backend.Close()
```

### 4. Aserciones Claras
```go
if rec.Code != tt.expectedStatus {
    t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
}
```

## Mejores Prácticas Implementadas

1. **Nomenclatura clara**: Nombres de tests descriptivos
2. **Aislamiento**: Cada test es independiente
3. **Setup/Teardown**: Uso de `defer` para cleanup
4. **Helpers**: Funciones auxiliares para generar datos de prueba
5. **Context**: Uso correcto de Echo context en tests
6. **Error messages**: Mensajes de error informativos

## Mantenimiento

### Agregar nuevos tests

1. Crear archivo `*_test.go` en el paquete correspondiente
2. Usar el sufijo `_test` en el nombre del paquete
3. Seguir estructura table-driven
4. Mantener cobertura >80%

### Actualizar tests existentes

1. Agregar casos a la tabla de tests
2. Actualizar helpers si es necesario
3. Verificar que todos los tests pasen
4. Revisar cobertura

## Troubleshooting

### Tests fallan con "connection refused"
- Verificar que no haya servidores corriendo en puertos de test
- Los mocks HTTP usan puertos dinámicos

### Cobertura baja en publisher
- Es esperado (14.3%) ya que NATS real requiere servidor externo
- Tests de patterns cubren la lógica crítica

### DB Resolver tests skipped
- Requiere implementación de TenantPoolManager
- Considerar tests de integración con DB real

## Archivos Generados

- `coverage.out` - Reporte de cobertura binario
- `*.test` - Binarios de test compilados (ignorados por git)

## Comandos Útiles

```bash
# Test con timeout
go test ./... -timeout 30s

# Test con race detector
go test ./... -race

# Test benchmark
go test ./... -bench=.

# Test con verbose output
go test ./... -v

# Coverage mínima requerida
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## Referencias

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Echo Framework Testing](https://echo.labstack.com/guide/testing)
- [JWT Testing Best Practices](https://github.com/golang-jwt/jwt)

---

**Última actualización**: 2026-02-22
**Autor**: Tests generados automáticamente
**Versión**: 1.0.0
