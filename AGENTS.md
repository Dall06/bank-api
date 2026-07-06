# ADN del Proyecto y Reglas para IA (AI_RULES.md)

Este documento centraliza todas las convenciones, reglas estrictas y lecciones aprendidas durante el desarrollo de este proyecto. Cualquier IA o agente (como Gemini, Cursor o Claude) debe leer y acatar estas directrices antes de tocar el código.

## 1. Patrones Arquitectónicos
* **Arquitectura Hexagonal (Puertos y Adaptadores)**: Innegociable. La lógica de negocio (casos de uso y dominio) no debe conocer de HTTP, bases de datos ni colas. Todo va detrás de interfaces (puertos).
* **Monorepo Estricto**: Los microservicios viven juntos pero aislados en `srv/` y `cmd/`. Comparten librerías bajo `pkg/`.
* **Regla de Pkg**: Un paquete dentro del directorio `pkg/` NO PUEDE importar a otro paquete dentro del mismo `pkg/`. Si hay solapamiento, significa que la abstracción está mal diseñada.

## 2. Convenciones de Código Go (Golang)
* **KISS (Keep It Simple, Stupid)**: Priorizar la legibilidad. Si el código es muy "inteligente" o complejo de leer, hay que refactorizar.
* **NO ELSE (Retornos tempranos)**: Queda TERMINANTEMENTE PROHIBIDO el uso de bloques `else` cuando el bloque `if` ya contiene un retorno. 
  ```go
  // INCORRECTO
  if err != nil {
      return err
  } else {
      return result
  }
  
  // CORRECTO
  if err != nil {
      return err
  }
  return result
  ```
* **Table-Driven Tests**: Todas las pruebas unitarias y mocks deben implementarse usando el patrón de tablas (Table-Driven Tests) con casos de uso claros (`name`, `input`, `mockBehavior`, `wantError`). ¡No borrar los test files generados!
* **Manejo de Errores Transversales**: Siempre usar `pkg/errs` para regresar errores hacia la capa HTTP, permitiendo mapeos centralizados en el middleware. No regresar `echo.NewHTTPError` directo desde casos de uso.
* **Fire-and-Forget**: Para tareas lentas y no críticas para la transacción HTTP (ej. emitir a Kafka o limpiar caché Redis), usar Goroutines asíncronas con `context.Background()` para no bloquear al usuario con latencia de red.

## 3. Protocolos de Desarrollo para la IA
* **Medir Dos Veces, Cortar Una**: La IA debe razonar el impacto completo antes de proponer cambios en múltiples archivos. Primero entender el flujo completo.
* **Autorización Explícita**: Queda estrictamente prohibido que la IA ejecute comandos destructivos o escriba archivos críticos del proyecto en masa sin pedir permiso o sin avisar detalladamente al usuario humano.
* **El usuario es el Dueño del Clúster**: El levantamiento de la infraestructura (ej. `task up`, `docker-compose`) los corre el usuario, a menos que el usuario delegue explícitamente a la IA que lo haga.
* **Logs al Rescate**: Frente a un error 500, la primera acción de la IA siempre debe ser mirar los logs de Docker del contenedor respectivo (`docker-compose logs <service>`) antes de inventar teorías.

## 4. Convenciones Operativas
* **Git**: Usar *Conventional Commits* en español (ej. `feat: agregar idempotencia en Redis`).
* **Go Version**: Mantener paridad absoluta (Go 1.25) en `go.mod`, `Dockerfile`, `.github/workflows` y entorno local. 
* **Variables de Entorno**: Cualquier variable nueva requerida por un microservicio debe documentarse en `docker-compose.yml` y `.env.example`.
